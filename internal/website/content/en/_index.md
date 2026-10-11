---
title: Go Micro
description: Build services in Go. Connect them, give agents tools to use them, and coordinate work with workflows.
params:
  body_class: micro-home
---

<div class="micro-landing">
<section class="micro-intro" aria-labelledby="intro-title">
<div>
<p class="micro-eyebrow">The Go framework for services and agents</p>
<h1 id="intro-title">Build services.<br>Put agents to work.</h1>
<p class="micro-lead">Turn what your software does into something an agent can use. Go Micro connects your services, exposes their capabilities as tools, and gives you a framework to build on.</p>
<div class="micro-actions">
<a class="micro-button" href="/docs/quickstart.html">Build your first project <span aria-hidden="true">→</span></a>
<a href="https://github.com/micro/go-micro">Explore the code</a>
</div>
<p class="micro-caption">Open source · Apache 2.0 · Built in Go</p>
</div>
<div class="micro-start" aria-labelledby="start-title">
<div class="micro-start-heading"><h2 id="start-title">Start a Go project</h2></div>
<pre aria-label="Create a Go Micro project"><code>mkdir my-app &amp;&amp; cd my-app
go mod init example.com/my-app
go get go-micro.dev/v6@master</code></pre>
<p>Write a service, give an agent access to it, and coordinate work with workflows.</p>
<p class="micro-caption">Requires Go 1.26+. The guides use the development version. <a href="/docs/quickstart.html">Get started →</a></p>
</div>
</section>

<section class="micro-foundation" aria-labelledby="foundation-title">
<div>
<p class="micro-eyebrow">A service framework you can build on</p>
<h2 id="foundation-title">Write the capability.<br>Let the framework connect it.</h2>
<p>Go Micro handles service discovery, RPC, and publish/subscribe so your services can find each other, make requests, and react to events.</p>
<p>Storage, configuration, and infrastructure integrations use replaceable Go interfaces. Start locally and choose the components your deployment needs.</p>
<p>Use services on their own, give agents access to them as tools, or compose them into workflows.</p>
<a href="/docs/reference.html">Explore the framework →</a>
</div>
<dl class="micro-building-blocks">
<div><dt>Services</dt><dd>Implement your application’s capabilities and own its data. Call them from Go or through a gateway.<a href="/docs/services.html">Build services →</a></dd></div>
<div><dt>Agents</dt><dd>Combine a model, conversation history, and tools to carry out requests. Define and extend their behaviour in Go.<a href="/docs/agents.html">Build an agent →</a></dd></div>
<div><dt>Workflows</dt><dd>Define repeatable steps around service calls and agent decisions, with recorded progress.<a href="/docs/workflows.html">Compose a workflow →</a></dd></div>
</dl>
</section>

<section class="micro-finish" aria-labelledby="finish-title">
<div><h2 id="finish-title">Start small. Build something useful.</h2><p>One project, your code, and a framework that grows with it.</p></div>
<a class="micro-button" href="/docs/quickstart.html">Get started <span aria-hidden="true">→</span></a>
</section>
</div>
