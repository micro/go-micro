#!/usr/bin/env python3
"""Exercise the installed CLI through a real terminal, using a loopback model.

Usage: python3 internal/harness/chat/run.py /absolute/path/to/micro
Only model responses are scripted. File tools, approvals, HTTP streaming,
settings, history, command execution, and process restarts are real.
No API keys or third-party network services are used.
"""
import fcntl
import http.server
import json
import os
from pathlib import Path
import pty
import re
import select
import struct
import subprocess
import sys
import tempfile
import termios
import threading
import time

BINARY = str(Path(sys.argv[1]).resolve())
ANSI = re.compile(r"\x1b\[[0-?]*[ -/]*[@-~]")
REQUESTS = []
ERRORS = []
RELEASE = threading.Event()


class Provider(http.server.BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(b'{"data":[{"id":"test-model"}]}')

    def do_POST(self):
        try:
            request = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
            REQUESTS.append(request)
            assert self.path == "/v1/chat/completions", self.path
            if self.headers.get("Authorization") != "Bearer test-key":
                self.send_response(401)
                self.end_headers()
                self.wfile.write(b'{"error":{"message":"invalid test key"}}')
                return
            messages = request["messages"]
            last = max(i for i, m in enumerate(messages) if m["role"] == "user")
            prompt = messages[last]["content"]
            results = [m["content"] for m in messages[last + 1:] if m["role"] == "tool"]
            call = None
            reply = "Reply received."
            if prompt == "Update the greeting":
                assert "Use the greeting file" in json.dumps(messages), "AGENTS.md missing"
                steps = [
                    ("workspace_read", {"path": "greeting.txt"}),
                    ("workspace_write", {"path": "greeting.txt", "content": "hello\n"}),
                    ("workspace_exec", {"command": "test \"$(cat greeting.txt)\" = hello && echo verified"}),
                ]
                if len(results) < len(steps):
                    call = steps[len(results)]
                else:
                    assert "before" in results[0], results
                    assert "verified" in results[-1], results
                    reply = "Updated and verified the greeting."
            elif prompt == "Try a denied write":
                if not results:
                    call = ("workspace_write", {"path": "denied.txt", "content": "must not exist"})
                else:
                    assert "denied" in results[0].lower(), results
                    reply = "The write was denied."
            elif prompt == "Hold this request":
                self.send_response(200)
                self.send_header("Content-Type", "text/event-stream")
                self.end_headers()
                self.wfile.write(b'data: {"choices":[{"index":0,"delta":{"content":"Holding request"}}]}\n\n')
                self.wfile.flush()
                assert RELEASE.wait(10), "queued input did not arrive"
                self.wfile.write(b'data: {"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}\n\ndata: [DONE]\n\n')
                return
            elif prompt == "Wait for cancellation":
                self.send_response(200)
                self.send_header("Content-Type", "text/event-stream")
                self.end_headers()
                self.wfile.write(b'data: {"choices":[{"index":0,"delta":{"content":"Waiting for cancellation"}}]}\n\n')
                self.wfile.flush()
                # Client cancellation closes the HTTP stream. Do not hold up server shutdown.
                time.sleep(3)
                return
            elif prompt == "Remember our change":
                assert "Updated and verified the greeting." in json.dumps(messages)
                reply = "The greeting is hello."
            delta = {"content": reply}
            reason = "stop"
            if call:
                name, args = call
                delta = {"tool_calls": [{"index": 0, "id": f"call-{len(results)}", "type": "function", "function": {"name": name, "arguments": json.dumps(args)}}]}
                reason = "tool_calls"
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.end_headers()
            for value in [delta, {}]:
                chunk = {"choices": [{"index": 0, "delta": value, "finish_reason": None if value else reason}]}
                self.wfile.write(("data: " + json.dumps(chunk) + "\n\n").encode())
                self.wfile.flush()
            self.wfile.write(b"data: [DONE]\n\n")
        except (BrokenPipeError, ConnectionResetError):
            pass
        except Exception as error:
            ERRORS.append(str(error))
            self.send_error(500, "mock assertion failed")


class Terminal:
    def __init__(self, project, profile, *args, extra_env=None):
        self.master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 30, 100, 0, 0))
        # Do not inherit provider credentials, config, or a developer's registry.
        self.env = {"PATH": os.environ["PATH"], "HOME": str(profile), "TERM": "xterm-256color", "NO_COLOR": "1", "MICRO_REGISTRY": "memory"}
        self.env.update(extra_env or {})
        self.process = subprocess.Popen([BINARY, "chat", *args], cwd=project, env=self.env, stdin=slave, stdout=slave, stderr=slave)
        os.close(slave)
        self.output = ""
        self.position = 0

    def send(self, text):
        os.write(self.master, text.encode() + b"\r")

    def expect(self, text, timeout=15):
        end = time.monotonic() + timeout
        while time.monotonic() < end:
            clean = ANSI.sub("", self.output)
            index = clean.find(text, self.position)
            if index >= 0:
                self.position = index + len(text)
                return
            if select.select([self.master], [], [], 0.1)[0]:
                try:
                    self.output += os.read(self.master, 65536).decode(errors="replace")
                except OSError:
                    break
        raise AssertionError(f"Expected {text!r}; terminal output:\n{ANSI.sub('', self.output)}")

    def close(self):
        if self.process.poll() is None:
            self.process.terminate()
        try:
            self.process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            self.process.kill()
            self.process.wait()
        os.close(self.master)

    def exit(self):
        self.send("/exit")
        assert self.process.wait(timeout=5) == 0


server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Provider)
threading.Thread(target=server.serve_forever, daemon=True).start()
endpoint = f"http://127.0.0.1:{server.server_port}"
terminals = []
try:
    with tempfile.TemporaryDirectory(prefix="micro-chat-") as directory:
        root = Path(directory)
        project, profile = root / "project", root / "profile"
        project.mkdir()
        profile.mkdir()
        (project / "AGENTS.md").write_text("Use the greeting file when testing this project.\n")
        (project / "greeting.txt").write_text("before\n")

        def boot(*args, extra_env=None):
            terminal = Terminal(project, profile, *args, extra_env=extra_env)
            terminals.append(terminal)
            return terminal

        t = boot()
        t.expect("Choose a model provider")
        t.send("999")
        t.expect("Selection out of range")
        t.send("not-a-provider")
        t.expect("Unknown provider")
        t.send("openai")
        t.expect("API key:")
        t.send("test-key")
        t.expect("micro >")
        t.send("/help")
        t.expect("Conversation")
        t.expect("Background work")
        t.expect("micro >")
        assert "test-key" not in t.output, "API key was echoed"
        t.send(f"/provider openai {endpoint}")
        t.expect("API key:")
        t.send("test-key")
        t.expect("Provider: openai")
        t.send("/model")
        t.expect("test-model")
        t.send("1")
        t.expect("Model: test-model")
        t.send("Update the greeting")
        t.expect("Approve workspace_write?")
        assert (project / "greeting.txt").read_text() == "before\n"
        t.send("/approve")
        t.expect("Approve workspace_exec?")
        t.send("/approve")
        t.expect("Updated and verified the greeting.")
        t.expect("micro >")
        assert (project / "greeting.txt").read_text() == "hello\n"
        t.send("Try a denied write")
        t.expect("Approve workspace_write?")
        t.send("/deny")
        t.expect("The write was denied.")
        t.expect("micro >")
        assert not (project / "denied.txt").exists()
        t.exit()
        print("PASS: first launch, hidden key, model selection, real read/write/exec, approvals and denial")

        t = boot()
        t.expect("Model:   openai / test-model")
        t.expect("Updated and verified the greeting.")
        t.expect("micro >")
        t.send("Remember our change")
        t.expect("The greeting is hello.")
        t.expect("micro >")
        t.send("/login")
        t.expect("API key:")
        t.send("wrong-key")
        t.expect("Key saved.")
        t.send("Check credentials")
        t.expect("invalid test key")
        t.expect("micro >")
        t.send("/login")
        t.expect("API key:")
        t.send("test-key")
        t.expect("Key saved.")
        t.send("Retry after login")
        t.expect("Reply received.")
        t.expect("micro >")
        t.send("Wait for cancellation")
        t.expect("Waiting for cancellation")
        t.send("/stop")
        t.expect("Stopped.")
        t.expect("micro >")
        t.send("After cancellation")
        t.expect("Reply received.")
        t.expect("micro >")
        t.send("Hold this request")
        t.expect("Holding request")
        t.send("Queued followup")
        t.expect("Queued (1)")
        RELEASE.set()
        t.expect("Reply received.")
        t.expect("micro >")
        t.send("/paste")
        t.expect("Enter multiple lines")
        t.send("First line")
        t.send("Second line")
        t.send("/send")
        t.expect("Reply received.")
        t.expect("micro >")
        assert REQUESTS[-1]["messages"][-1]["content"] == "First line\nSecond line"
        t.send("/new")
        t.expect("Session ")
        t.send("Fresh conversation")
        t.expect("Reply received.")
        t.expect("micro >")
        assert "Updated and verified the greeting." not in json.dumps(REQUESTS[-1])
        t.exit()
        print("PASS: restart restores history and settings; bad-key recovery; cancellation; queued and multiline input; new session isolation")

        clean_project = root / "clean"
        clean_project.mkdir()
        env = {"PATH": os.environ["PATH"], "HOME": str(profile), "MICRO_REGISTRY": "memory"}
        result = subprocess.run([BINARY, "chat", "--prompt", "Hello"], cwd=clean_project, env=env, capture_output=True, text=True, timeout=10)
        assert result.returncode != 0 and "no API key configured" in result.stderr + result.stdout
        assert "API key:" not in result.stderr + result.stdout
        t = Terminal(clean_project, profile, "--prompt", "Hello")
        terminals.append(t)
        t.expect("no API key configured")
        assert t.process.wait(timeout=5) != 0
        assert "API key:" not in t.output
        env["GROQ_API_KEY"] = "test-key"
        t = Terminal(clean_project, profile, extra_env=env)
        terminals.append(t)
        t.expect("Model:   groq /")
        t.exit()
        print("PASS: noninteractive missing credentials fail promptly; provider inferred from its environment key")
        # The default discovery backend must also reach onboarding on a clean start.
        default_project = root / "default"
        default_project.mkdir()
        t = Terminal(default_project, profile, extra_env={"MICRO_REGISTRY": "mdns"})
        terminals.append(t)
        t.expect("Choose a model provider")
        t.send("")
        assert t.process.wait(timeout=5) == 0
        print("PASS: default mDNS startup reaches onboarding; empty selection cancels cleanly")
        assert not ERRORS, ERRORS
finally:
    for terminal in terminals:
        terminal.close()
    server.shutdown()
