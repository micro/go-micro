#!/usr/bin/env python3
"""Run the documented service through RPC, HTTP, MCP and a Go agent.

Usage: python3 internal/harness/services.py /path/to/micro /path/to/nats-server
Requires Go and a NATS server binary. Uses only loopback servers and a scripted
model; no provider credentials. Builds the Go examples directly from the docs.
"""
import contextlib
import http.server
import json
import os
from pathlib import Path
import socket
import select
import subprocess
import sys
import tempfile
import threading
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
MICRO, NATS = map(lambda p: str(Path(p).resolve()), sys.argv[1:3])


def port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def post(url, body, token=None):
    headers = {"Content-Type": "application/json", "Accept": "application/json, text/event-stream"}
    if token:
        headers["Authorization"] = "Bearer " + token
    req = urllib.request.Request(url, json.dumps(body).encode(), headers)
    with urllib.request.urlopen(req, timeout=10) as r:
        return json.load(r)


def rpc(method, params=None):
    return {"jsonrpc": "2.0", "id": 1, "method": method, "params": params or {}}


def wait_port(number, process):
    for _ in range(100):
        if process.poll() is not None:
            raise RuntimeError(f"process exited: {process.args}")
        try:
            with socket.create_connection(("127.0.0.1", number), timeout=.1):
                return
        except OSError:
            time.sleep(.05)
    raise TimeoutError(f"port {number}")


class Model(http.server.BaseHTTPRequestHandler):
    calls = 0

    def log_message(self, *args):
        pass

    def do_POST(self):
        request = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        results = [m for m in request["messages"] if m["role"] == "tool"]
        if results:
            assert "Review the release" in json.dumps(results), results
            Model.calls += 1
            message = {"role": "assistant", "content": "Review the release; Update the documentation."}
        else:
            name = next(t["function"]["name"] for t in request["tools"] if "Notes" in t["function"]["name"])
            message = {"role": "assistant", "content": "", "tool_calls": [{"id": "notes01", "type": "function", "function": {"name": name, "arguments": "{}"}}]}
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps({"choices": [{"message": message}]}).encode())


with tempfile.TemporaryDirectory(prefix="micro-services-") as tmp, contextlib.ExitStack() as stack:
    work = Path(tmp)
    env = {k: v for k, v in os.environ.items() if not k.startswith("MICRO_") and not k.endswith("API_KEY")}
    env.update(HOME=tmp, XDG_CONFIG_HOME=tmp, GOFLAGS="-buildvcs=false -mod=mod")
    # Reuse build caches; the CLI's home/config remain isolated.
    for name in ("GOPATH", "GOCACHE"):
        env[name] = subprocess.check_output(["go", "env", name], text=True).strip()
    env["GOTOOLCHAIN"] = "go1.26.9"
    reg, api, mcp = port(), port(), port()
    env.update(MICRO_REGISTRY="nats", MICRO_REGISTRY_ADDRESS=f"127.0.0.1:{reg}")

    def start(args):
        log = open(work / f"process-{len(processes)}.log", "w+")
        p = subprocess.Popen(args, cwd=work, env=env, stdout=log, stderr=log)
        processes.append((p, log))
        return p

    processes = []

    def cleanup():
        for p, log in reversed(processes):
            p.terminate()
            try:
                p.wait(timeout=5)
            except subprocess.TimeoutExpired:
                p.kill()
                p.wait()
        if sys.exc_info()[0]:
            for p, log in processes:
                log.seek(0)
                print(log.read(), file=sys.stderr)
        for p, log in processes:
            log.close()

    stack.callback(cleanup)
    nats = start([NATS, "-a", "127.0.0.1", "-p", str(reg)])
    wait_port(reg, nats)
    (work / "go.mod").write_text(f"module example.com/journey\n\ngo 1.26.0\nrequire go-micro.dev/v6 v6.0.0\nreplace go-micro.dev/v6 => {ROOT}\n")

    def build(doc, output):
        code = (ROOT / "internal/website/content/en/docs" / doc).read_text().split("```go\n")[1].split("```")[0]
        code = code.replace('micro "go-micro.dev/v6"', 'micro "go-micro.dev/v6"\n _ "go-micro.dev/v6/cmd/defaults"')
        if output == "assistant":
            code = code.replace('micro "go-micro.dev/v6"', 'micro "go-micro.dev/v6"\n "go-micro.dev/v6/agent"\n "go-micro.dev/v6/client"\n "go-micro.dev/v6/registry"\n "go-micro.dev/v6/registry/nats"')
            code = code.replace('agent := micro.NewAgent("assistant",', 'reg := nats.NewNatsRegistry(registry.Addrs(os.Getenv("MICRO_REGISTRY_ADDRESS")))\n agent := micro.NewAgent("assistant",\n agent.Address("127.0.0.1:0"),\n agent.WithRegistry(reg),\n agent.WithClient(client.NewClient(client.Registry(reg))),')
        (work / "main.go").write_text(code)
        subprocess.run(["go", "build", "-o", str(work / output), "."], cwd=work, env=env, check=True)

    build("quickstart.md", "notes")
    notes = start([str(work / "notes"), "--server_address", "127.0.0.1:0"])
    for _ in range(30):
        call = subprocess.run([MICRO, "call", "notes", "Notes.List", "{}"], env=env, cwd=work, capture_output=True, text=True)
        if call.returncode == 0:
            break
        time.sleep(.1)
    assert call.returncode == 0 and "Review the release" in call.stdout, call.stderr
    print("PASS: documented Go service via real RPC and NATS discovery", flush=True)

    gateway = start([MICRO, "gateway", "--address", f"127.0.0.1:{api}", "--mcp-address", f"127.0.0.1:{mcp}", "--mcp-tool", "notes.Notes.List", "--auth", "--auth-token", "local-test-token"])
    wait_port(api, gateway)
    wait_port(mcp, gateway)
    url = f"http://127.0.0.1:{mcp}/mcp"
    for target, body in [(url, rpc("initialize")), (f"http://127.0.0.1:{api}/api/notes/Notes/List", {})]:
        try:
            post(target, body)
            raise AssertionError("unauthenticated request accepted")
        except urllib.error.HTTPError as e:
            assert e.code == 401, e.code
    assert "Review the release" in json.dumps(post(f"http://127.0.0.1:{api}/api/notes/Notes/List", {}, "local-test-token"))
    initialized = post(url, rpc("initialize", {"protocolVersion": "2025-03-26", "capabilities": {}, "clientInfo": {"name": "journey", "version": "1"}}), "local-test-token")
    assert "result" in initialized, initialized
    tools = post(url, rpc("tools/list"), "local-test-token")["result"]["tools"]
    assert [t["name"] for t in tools] == ["notes.Notes.List"], tools
    result = post(url, rpc("tools/call", {"name": "notes.Notes.List", "arguments": {}}), "local-test-token")
    assert "Review the release" in json.dumps(result), result
    denied = post(url, rpc("tools/call", {"name": "micro_registry_list", "arguments": {}}), "local-test-token")
    assert "error" in denied or denied.get("result", {}).get("isError"), denied
    print("PASS: HTTP and MCP initialize/list/call; shared token; excluded tools rejected", flush=True)

    requests = [rpc("initialize", {"protocolVersion": "2025-03-26", "capabilities": {}, "clientInfo": {"name": "journey", "version": "1"}}), rpc("tools/list"), rpc("tools/call", {"name": "notes.Notes.List", "arguments": {}})]
    stdio = subprocess.Popen([MICRO, "mcp", "serve", "--tool", "notes.Notes.List"], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, env=env, cwd=work, text=True)
    replies = []
    try:
        for request in requests:
            stdio.stdin.write(json.dumps(request) + "\n")
            stdio.stdin.flush()
            assert select.select([stdio.stdout], [], [], 10)[0], "MCP response timed out"
            replies.append(json.loads(stdio.stdout.readline()))
            if request["method"] == "initialize":
                stdio.stdin.write(json.dumps({"jsonrpc": "2.0", "method": "notifications/initialized"}) + "\n")
                stdio.stdin.flush()
        assert "Review the release" in json.dumps(replies[-1]), replies
        assert [t["name"] for t in replies[1]["result"]["tools"]] == ["notes.Notes.List"]
    finally:
        stdio.stdin.close()
        try:
            stdio.wait(timeout=5)
        except subprocess.TimeoutExpired:
            stdio.kill()
            stdio.wait()
    print("PASS: independent stdio MCP client; clean protocol output", flush=True)

    occupied = subprocess.run([MICRO, "gateway", "--address", "127.0.0.1:0", "--mcp-address", f"127.0.0.1:{reg}"], env=env, cwd=work, capture_output=True, text=True, timeout=10)
    assert occupied.returncode != 0 and "address already in use" in occupied.stderr + occupied.stdout, (occupied.stdout, occupied.stderr)
    print("PASS: occupied MCP listener fails promptly instead of leaving a partial gateway", flush=True)

    model = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Model)
    stack.callback(model.server_close)
    stack.callback(model.shutdown)
    threading.Thread(target=model.serve_forever, daemon=True).start()
    env.update(OPENAI_API_KEY="test-key", MICRO_AI_BASE_URL=f"http://127.0.0.1:{model.server_port}")
    local = subprocess.run([MICRO, "chat", "--provider", "openai", "--yes", "--prompt", "What notes do I have?"], env=env, cwd=work, capture_output=True, text=True, timeout=20)
    assert local.returncode == 0 and "Review the release" in local.stdout, (local.stdout, local.stderr)
    print("PASS: local micro chat calls the running service (scripted model)", flush=True)
    build("agents.md", "assistant")
    assistant = start([str(work / "assistant")])
    for _ in range(30):
        listing = subprocess.run([MICRO, "list", "services"], env=env, cwd=work, capture_output=True, text=True)
        if "assistant" in listing.stdout:
            break
        time.sleep(.1)
    remote_env = {k: v for k, v in env.items() if k not in ("OPENAI_API_KEY", "MICRO_AI_BASE_URL")}
    remote = subprocess.run([MICRO, "chat", "--prompt", "What notes do I have?"], env=remote_env, cwd=work, capture_output=True, text=True, timeout=20)
    assert remote.returncode == 0 and "Review the release" in remote.stdout, (remote.stdout, remote.stderr)
    assert Model.calls == 2, Model.calls
    print("PASS: documented custom Go agent; micro chat connects without client credentials", flush=True)
