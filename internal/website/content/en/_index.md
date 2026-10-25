---
title: Go Micro
description: A Go framework for services and agents. RPC, service discovery, messaging, storage, and configuration for distributed applications.
params:
  body_class: micro-home
---

<div class="micro-landing">
<section class="micro-intro" aria-labelledby="intro-title">
<div>
<h1 id="intro-title">A Go framework for<br>services and agents.</h1>
<p class="micro-lead">Build distributed applications with RPC, service discovery, messaging, storage, and configuration. Create agents that use your services as tools.</p>
<div class="micro-actions">
<a class="micro-button" href="/docs/quickstart.html">Get Started <span aria-hidden="true">→</span></a>
<a href="https://github.com/micro/go-micro">GitHub</a>
</div>
<p class="micro-caption">Open source · Apache 2.0</p>
</div>
<figure class="micro-example" aria-labelledby="example-title">
<figcaption id="example-title"><span>Services + agents</span><span>Go</span></figcaption>
<pre><code>service := micro.NewService("notes")
service.Init()

if err := service.Handle(new(Notes)); err != nil {
    log.Fatal(err)
}

if err := service.Handle(
    micro.NewAgent("notes"),
); err != nil {
    log.Fatal(err)
}

if err := service.Run(); err != nil {
    log.Fatal(err)
}</code></pre>
<p>One service, available through RPC and an agent.</p>
<a href="/docs/agents.html#add-an-agent-to-a-service">Add an agent to your service →</a>
</figure>
</section>

<section class="micro-journey" aria-label="Explore the framework">
<ul class="micro-steps">
<li>
<h2>Build services</h2>
<p>Write application capabilities in Go. Connect services through RPC and messaging.</p>
<a href="/docs/quickstart.html">Services →</a>
</li>
<li>
<h2>Build agents</h2>
<p>Combine models with your services to answer questions and perform work.</p>
<a href="/docs/agents.html">Agents →</a>
</li>
<li>
<h2>Connect other agents</h2>
<p>Expose your services through MCP for use by existing agents and tools.</p>
<a href="/docs/guides/services-with-claude-code.html">MCP →</a>
</li>
</ul>
<div class="micro-follow">
<p>Each component uses a Go interface with a default implementation you can replace. Choose the components your application needs.</p>
<a href="/docs/reference.html">Framework reference →</a>
</div>
</section>
</div>
