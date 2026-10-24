---
title: Go Micro
description: Build services in Go. Connect them and give agents tools to use them.
params:
  body_class: micro-home
---

<div class="micro-landing">
<section class="micro-intro" aria-labelledby="intro-title">
<div>
<p class="micro-eyebrow">The Go framework for services and agents</p>
<h1 id="intro-title">Build services.<br>Put agents to work.</h1>
<p class="micro-lead">Build services in Go. Call them from your applications, connect agents through MCP, or write your own agents.</p>
<div class="micro-actions">
<a class="micro-button" href="/docs/quickstart.html">Get started <span aria-hidden="true">→</span></a>
<a href="https://github.com/micro/go-micro">Explore the code</a>
</div>
<p class="micro-caption">Open source · Apache 2.0 · Built in Go</p>
</div>
<div class="micro-start" aria-labelledby="start-title">
<div class="micro-start-heading"><h2 id="start-title">A running service</h2></div>
<pre aria-label="Call the notes service from the getting-started guide"><code>$ micro call notes Notes.List '{}'
{
  "notes": [
    "Review the release",
    "Update the documentation"
  ]
}</code></pre>
<p>Expose the same operation through MCP, use it in chat, or call it from your Go code.</p>
<p class="micro-caption"><a href="/docs/quickstart.html">Build and run this service →</a></p>
</div>
</section>

<section class="micro-foundation" aria-labelledby="foundation-title">
<div>
<p class="micro-eyebrow">A service framework you can build on</p>
<h2 id="foundation-title">Write the capability.<br>Let the framework connect it.</h2>
<p>Go Micro handles service discovery, RPC, and publish/subscribe so your services can find each other, make requests, and react to events.</p>
<p>Storage, configuration, and infrastructure integrations use replaceable Go interfaces. Start locally and choose the components your deployment needs.</p>
<p>Use services on their own or <a href="/docs/agents.html">build your own agent</a>.</p>
<a href="/docs/reference.html">Explore the framework →</a>
</div>
<dl class="micro-building-blocks">
<div><dt>Build a service</dt><dd>Write Go methods, run the service, and call it over RPC. No model or agent needed.<a href="/docs/quickstart.html">Build and run →</a></dd></div>
<div><dt>Expose it to agents</dt><dd>Already have services and an agent? Use the gateway to expose operations through MCP.<a href="/docs/guides/services-with-claude-code.html">Connect Claude Code →</a></dd></div>
<div><dt>Chat with it</dt><dd>Use Go Micro’s terminal agent to call your services. No gateway required.<a href="/docs/guides/chat-with-services.html">Start chatting →</a></dd></div>
</dl>
</section>

<section class="micro-finish" aria-labelledby="finish-title">
<div><h2 id="finish-title">Start small. Build something useful.</h2><p>One project, your code, and a framework that grows with it.</p></div>
<a class="micro-button" href="/docs/quickstart.html">Get started <span aria-hidden="true">→</span></a>
</section>
</div>
