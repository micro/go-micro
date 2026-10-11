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
<div class="micro-start-heading"><h2 id="start-title">Start with a conversation</h2><span>01</span></div>
<p>Install the CLI, open a project, and describe what you want to build.</p>
<pre aria-label="Install and start micro chat"><code>go install go-micro.dev/v6/cmd/micro@master
export PATH="$(go env GOPATH)/bin:$PATH"
mkdir my-app &amp;&amp; cd my-app
micro chat</code></pre>
<p>Choose your model provider. The agent can read and change your project, run commands, and call your services. You approve file changes, commands, and service actions.</p>
<p class="micro-caption">Requires Go 1.26+ and access to a model. This chat experience is on the development branch. <a href="/docs/quickstart.html">Setup and prerequisites →</a></p>
</div>
</section>

<section class="micro-journey" aria-labelledby="journey-title">
<div class="micro-section-heading">
<p class="micro-eyebrow">From an idea to a capability you can use</p>
<h2 id="journey-title">Build it. Use it. Make it yours.</h2>
<p>Start with the work you need done. A notes service is one small way to try the whole journey.</p>
</div>
<ol class="micro-steps">
<li><span class="micro-step-number">01 / Build</span><h3>Describe the capability</h3><p>Ask the agent to build a service. Review its actions as it creates the source, compiles it, and starts it locally.</p><blockquote>Build a service named notes that saves and lists notes. Store them persistently.</blockquote></li>
<li><span class="micro-step-number">02 / Use</span><h3>Give it real work</h3><p>Once the service is running, send a new request. The agent discovers its endpoints as tools and calls them.</p><blockquote>Save a note: the launch review is on Friday. Then list my notes.</blockquote></li>
<li><span class="micro-step-number">03 / Extend</span><h3>Keep building</h3><p>Read the generated Go code. Change it yourself or ask the agent to extend it. Run the service independently when you leave chat.</p><blockquote>Add search to the notes service, then verify it finds my launch note.</blockquote></li>
</ol>
<div class="micro-follow"><p>You own the Go source and can run the services independently. Other applications can call the same capabilities.</p><a href="/docs/quickstart.html">Follow the complete walkthrough →</a></div>
</section>

<section class="micro-foundation" aria-labelledby="foundation-title">
<div>
<p class="micro-eyebrow">A service framework you can build on</p>
<h2 id="foundation-title">Write the capability.<br>Let the framework connect it.</h2>
<p>Go Micro handles service discovery, RPC, and publish/subscribe so your services can find each other, make requests, and react to events.</p>
<p>Storage, configuration, and infrastructure integrations use replaceable Go interfaces. Start locally and choose the components your deployment needs.</p>
<p>Use services on their own, give agents access to them as tools, or compose them into workflows.</p>
<a href="/docs/architecture.html">Explore the framework →</a>
</div>
<dl class="micro-building-blocks">
<div><dt>Services</dt><dd>Implement your application’s capabilities and own its data. Call them from Go or through a gateway.<a href="/docs/client-server.html">Build services →</a></dd></div>
<div><dt>Agents</dt><dd>Combine a model, conversation history, and tools to carry out requests. Define and extend their behaviour in Go.<a href="/docs/guides/your-first-agent.html">Build an agent →</a></dd></div>
<div><dt>Workflows</dt><dd>Define repeatable steps around service calls and agent decisions, with recorded progress.<a href="/docs/guides/agents-and-workflows.html">Compose a workflow →</a></dd></div>
</dl>
</section>

<section class="micro-finish" aria-labelledby="finish-title">
<div><h2 id="finish-title">Start small. Build something useful.</h2><p>One project, your code, and a framework that grows with it.</p></div>
<a class="micro-button" href="/docs/quickstart.html">Get started <span aria-hidden="true">→</span></a>
</section>
</div>
