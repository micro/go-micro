---
title: Go Micro
description: Build Go services, run them locally, and use their handlers as agent tools.
params:
  body_class: micro-home
---

<div class="micro-landing">
<section class="micro-intro" aria-labelledby="intro-title">
<div>
<p class="micro-eyebrow">The Go framework for services and agents</p>
<h1 id="intro-title">Build services.<br>Chat with them.</h1>
<p class="micro-lead">Write services in Go. Add an agent that uses their handlers as tools. Call the same service from code or a conversation.</p>
<div class="micro-actions">
<a class="micro-button" href="/docs/agents.html">Get started <span aria-hidden="true">→</span></a>
<a href="https://github.com/micro/go-micro">Explore the code</a>
</div>
<p class="micro-caption">Open source · Apache 2.0 · Built in Go</p>
</div>
<div class="micro-start" aria-labelledby="start-title">
<div class="micro-start-heading"><h2 id="start-title">New → Run → Chat</h2></div>
<pre aria-label="Create and run a service with an agent"><code>micro new weather --agent
cd weather
export OPENAI_API_KEY=your-key
micro run</code></pre>
<p>In another terminal:</p>
<pre aria-label="Chat with the running service"><code>micro chat weather</code></pre>
<p>Ask it to greet Alice. The starter returns “Hello Alice”. Add your own Go handlers to give it new capabilities.</p>
<p class="micro-caption">Use any <a href="/docs/guides/ai-provider-guide.html">supported model provider</a>. This path currently requires the CLI from <code>master</code>; <a href="/docs/agents.html">follow the setup guide →</a></p>
</div>
</section>

<section class="micro-foundation" aria-labelledby="foundation-title">
<div>
<p class="micro-eyebrow">Services at the core</p>
<h2 id="foundation-title">Your handlers. Your agent.</h2>
<p>Services provide the operations. Agents use them as tools. Go Micro handles RPC, service discovery, messaging, storage, and configuration.</p>
<p>Each component uses a Go interface with a default implementation you can replace. Use the framework in your application; the CLI helps you build and run it.</p>
<p>Use services on their own or <a href="/docs/agents.html">build your own agent</a>.</p>
<a href="/docs/reference.html">Explore the framework →</a>
</div>
<dl class="micro-building-blocks">
<div><dt>Build a service</dt><dd>Write Go methods, run the service, and call it over RPC. No model or agent needed.<a href="/docs/quickstart.html">Build and run →</a></dd></div>
<div><dt>Expose it through MCP</dt><dd>Already have services and an agent? Use the gateway to expose operations through MCP.<a href="/docs/guides/services-with-claude-code.html">Connect Claude Code →</a></dd></div>
<div><dt>Chat with it</dt><dd>Attach an agent to the service and connect with <code>micro chat</code>. One service, with RPC and chat on the same server.<a href="/docs/agents.html">Add an agent →</a></dd></div>
</dl>
</section>

<section class="micro-finish" aria-labelledby="finish-title">
<div><h2 id="finish-title">New. Run. Chat.</h2><p>Start with a service. Add the capabilities your application needs.</p></div>
<a class="micro-button" href="/docs/agents.html">Get started <span aria-hidden="true">→</span></a>
</section>
</div>
