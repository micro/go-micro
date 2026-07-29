---
title: Go Micro
description: Go Micro - A Go microservices framework
params:
  body_class: td-navbar-links-all-active bg-pattern
---

{{% blocks/hero
  height="full td-below-navbar"
  color="dark-blue"
  pattern=true
%}}

<div class="d-flex justify-content-center align-items-center mb-5">
<div class="logo-container position-relative text-center" role="button">

<!-- Glow -->
<div class="logo-glow position-absolute top-0 start-0 w-100 h-100 rounded-circle"></div>

<svg width="300" height="300" viewBox="0 0 200 200" class="hexagon relative z-10">
<!-- Defs for gradients -->
<defs>
<linearGradient id="hexGradient" x1="0%" y1="0%" x2="100%" y2="100%">
<stop offset="0%" style="stop-color:#00ADD8;stop-opacity:1"></stop>
<stop offset="100%" style="stop-color:#5DC9E2;stop-opacity:1"></stop>
</linearGradient>
<linearGradient id="lineGradient" x1="0%" y1="0%" x2="100%" y2="100%">
<stop offset="0%" style="stop-color:#00ADD8;stop-opacity:0.8"></stop>
<stop offset="100%" style="stop-color:#00ADD8;stop-opacity:0.3"></stop>
</linearGradient>
</defs>
<!-- Outer Hexagon -->
<polygon points="100,10 185,55 185,145 100,190 15,145 15,55" fill="none" stroke="url(#hexGradient)" stroke-width="4" stroke-linecap="round" stroke-linejoin="round"></polygon>
<!-- Inner Network Lines -->
<line x1="100" y1="100" x2="100" y2="55" stroke="url(#lineGradient)" stroke-width="2" class="network-line"></line>
<line x1="100" y1="100" x2="142.5" y2="77.5" stroke="url(#lineGradient)" stroke-width="2" class="network-line" style="animation-delay: 0.2s"></line>
<line x1="100" y1="100" x2="142.5" y2="122.5" stroke="url(#lineGradient)" stroke-width="2" class="network-line" style="animation-delay: 0.4s"></line>
<line x1="100" y1="100" x2="100" y2="145" stroke="url(#lineGradient)" stroke-width="2" class="network-line" style="animation-delay: 0.6s"></line>
<line x1="100" y1="100" x2="57.5" y2="122.5" stroke="url(#lineGradient)" stroke-width="2" class="network-line" style="animation-delay: 0.8s"></line>
<line x1="100" y1="100" x2="57.5" y2="77.5" stroke="url(#lineGradient)" stroke-width="2" class="network-line" style="animation-delay: 1s"></line>
<!-- Cross connections for distributed system look -->
<line x1="100" y1="55" x2="142.5" y2="77.5" stroke="rgba(0,173,216,0.2)" stroke-width="1" class="network-line" style="animation-delay: 1.2s"></line>
<line x1="142.5" y1="77.5" x2="142.5" y2="122.5" stroke="rgba(0,173,216,0.2)" stroke-width="1" class="network-line" style="animation-delay: 1.3s"></line>
<line x1="142.5" y1="122.5" x2="100" y2="145" stroke="rgba(0,173,216,0.2)" stroke-width="1" class="network-line" style="animation-delay: 1.4s"></line>
<line x1="100" y1="145" x2="57.5" y2="122.5" stroke="rgba(0,173,216,0.2)" stroke-width="1" class="network-line" style="animation-delay: 1.5s"></line>
<line x1="57.5" y1="122.5" x2="57.5" y2="77.5" stroke="rgba(0,173,216,0.2)" stroke-width="1" class="network-line" style="animation-delay: 1.6s"></line>
<line x1="57.5" y1="77.5" x2="100" y2="55" stroke="rgba(0,173,216,0.2)" stroke-width="1" class="network-line" style="animation-delay: 1.7s"></line>
<!-- Central Node (Core) -->
<circle cx="100" cy="100" r="12" fill="#0D1117" stroke="#00ADD8" stroke-width="3" class="network-node"></circle>
<circle cx="100" cy="100" r="6" fill="#00ADD8" class="network-node pulse"></circle>
<!-- Outer Nodes (Microservices) -->
<circle cx="100" cy="55" r="6" fill="#0D1117" stroke="#00ADD8" stroke-width="2" class="network-node node-1"></circle>
<circle cx="142.5" cy="77.5" r="6" fill="#0D1117" stroke="#00ADD8" stroke-width="2" class="network-node node-2"></circle>
<circle cx="142.5" cy="122.5" r="6" fill="#0D1117" stroke="#00ADD8" stroke-width="2" class="network-node node-3"></circle>
<circle cx="100" cy="145" r="6" fill="#0D1117" stroke="#00ADD8" stroke-width="2" class="network-node node-4"></circle>
<circle cx="57.5" cy="122.5" r="6" fill="#0D1117" stroke="#00ADD8" stroke-width="2" class="network-node node-5"></circle>
<circle cx="57.5" cy="77.5" r="6" fill="#0D1117" stroke="#00ADD8" stroke-width="2" class="network-node node-6"></circle>
<!-- Data packets animation -->
<circle cx="100" cy="77.5" r="2" fill="#5DC9E2" opacity="0">
<animate attributeName="opacity" values="0;1;0" dur="2s" repeatCount="indefinite" begin="0s"></animate>
<animate attributeName="cy" values="100;55" dur="2s" repeatCount="indefinite" begin="0s"></animate>
</circle>
<circle cx="121.25" cy="88.75" r="2" fill="#5DC9E2" opacity="0">
<animate attributeName="opacity" values="0;1;0" dur="2s" repeatCount="indefinite" begin="0.3s"></animate>
<animate attributeName="cx" values="100;142.5" dur="2s" repeatCount="indefinite" begin="0.3s"></animate>
<animate attributeName="cy" values="100;77.5" dur="2s" repeatCount="indefinite" begin="0.3s"></animate>
</circle>
</svg>

<!-- Text -->
<div class="text-center mb-5">
    <h1 class="display-3 fw-bold mb-4">
        <span class="gradient-text">GoMicro</span></br>
        An Agent Harness for Go
    </h1>
    <p class="lead text-gray-custom mb-5 mx-auto">
        Build agents, services, and workflows on one runtime.
    </p>
</div>

</div>
</div>

<!-- prettier-ignore -->
<div class="td-cta-buttons my-5">
<a class="download-btn btn btn-go btn-lg d-inline-flex align-items-center gap-2" href="docs/">
Learn more
<i class="fa-solid fa-book-open-reader"></i>
</a>
<a class="btn btn-outline-go btn-lg d-inline-flex align-items-center gap-2"
href="{{% param github_project_repo %}}"
target="_blank" rel="noopener noreferrer">
Get the code
<i class="fa-brands fa-github"></i>
</a>
</div>
{{% blocks/link-down color="info" %}}
{{% /blocks/hero %}}

{{% blocks/section color="dark-blue" type="row" pattern=true padding="py-5" %}}
<div class="row g-4 justify-content-center">
<h2>Features</h2>
<p class="text-light mb-4">An agent harness and a service framework in one — agents, services, and flows on the same runtime, with the production pieces agents need.
</p>
</div>
<div class="row g-4 justify-content-center">
<div class="col-lg-4 col-md-6">
{{% elements/variant-card
    color="gradient"
    title="Agent Harness"
    subtitle="A model, memory, tools, plan/delegate, guardrails, and execution middleware around every agent."
%}}
<div class="p-3 display-3">🤖</div>
{{% /elements/variant-card %}}
</div>

<div class="col-lg-4 col-md-6">
{{% elements/variant-card
    color="gradient"
    title="Services as Tools"
    subtitle="Build services in Go or generate them from a prompt. Their endpoints become typed tools automatically."
%}}
<div class="p-3 display-3">🔧</div>
{{% /elements/variant-card %}}
</div>

<div class="col-lg-4 col-md-6">
{{% elements/variant-card
    color="gradient"
    title="Durable Workflows"
    subtitle="Use fixed, checkpointed code paths for deterministic work; hand off to agents when the path is dynamic."
%}}
<div class="p-3 display-3">⚙️</div>
{{% /elements/variant-card %}}
</div>

<div class="col-lg-4 col-md-6">
{{% elements/variant-card
    color="gradient"
    title="MCP Gateway"
    subtitle="Every service endpoint is automatically an AI-callable tool via the Model Context Protocol."
%}}
<div class="p-3 display-3">🌐</div>
{{% /elements/variant-card %}}
</div>

<div class="col-lg-4 col-md-6">
{{% elements/variant-card
    color="gradient"
    title="A2A Gateway"
    subtitle="Every agent is reachable over the Agent2Agent protocol — discovered and called by agents on any framework."
%}}
<div class="p-3 display-3">🤝</div>
{{% /elements/variant-card %}}
</div>

<div class="col-lg-4 col-md-6">
{{% elements/variant-card
    color="gradient"
    title="Pluggable Everything"
    subtitle="All abstractions are Go interfaces. Swap any component without changing your code."
%}}
<div class="p-3 display-3">🔌</div>
{{% /elements/variant-card %}}
</div>
</div>
{{% /blocks/section %}}

{{% blocks/section color="dark-blue" pattern=true padding="py-5" %}}
<div class="row align-items-center">
<div class="col-lg-6 p-4">
<img src="/images/generated/mcp-agent.jpg" alt="AI agent calling microservices via MCP" class="img-fluid rounded-4 shadow" />
</div>
<div class="col-lg-6 p-4">
<h2>From Prompt Loop to Operating Harness</h2>
<p>Tell it what you need. The AI designs services, generates an agent, and drops you into an interactive console. The harness gives that agent tools, memory, guardrails, and workflows so it can operate across services.</p>
<pre><code><span style="color:#ff7b72">$</span> <span style="color:#d2a8ff">micro run</span> <span style="color:#a5d6ff">--prompt "task management system"</code></br>
<code><span style="color:#ff7b72">&gt;</span> Create a project called Launch, add tasks, assign to Alice</code></pre>
<a class="btn btn-go mt-3" href="/blog/2026/06/05/introducing-micro-newagent/">Learn About Agents</a>
</div>
</div>
{{% /blocks/section %}}

{{% blocks/section color="dark-blue" pattern=true padding="py-5" %}}
<div class="row align-items-center">
<div class="col-lg-6 p-4">
<img src="/images/generated/architecture.jpg" alt="Go Micro architecture diagram" class="img-fluid rounded-4 shadow" />
</div>
<div class="col-lg-6 p-4 order-lg-first">
<h2>Distributed Systems for Agents</h2>
<p>Agents need the same substrate services do: discovery, RPC, events, state, auth, observability, and deployment. Swap any component without changing your code — go from mDNS to Consul, or HTTP to gRPC, with a single option.</p>
</div>
</div>
{{% /blocks/section %}}

{{% blocks/section color="dark-blue" pattern=true padding="py-5" %}}
<div class="row align-items-center">
<div class="col-lg-6 p-4">
<img src="/images/generated/developer-experience.jpg" alt="Terminal showing micro run and micro chat" class="img-fluid rounded-4 shadow" />
</div>
<div class="col-lg-6 p-4">
<h2>Developer Experience</h2>
<p><code>micro new</code> scaffolds a service or an agent — every endpoint is automatically an MCP tool. <code>micro run</code> starts everything with hot reload, an API gateway, and an interactive console, and <code>micro chat</code> talks to your agents and services from the terminal.</p>
</div>
</div>
{{% /blocks/section %}}

{{% blocks/section color="dark-blue" pattern=true padding="py-5" type="text-center" %}}
<h2>Sponsors</h2>
<p class="text-light mb-4">Go Micro is supported by companies building the future of AI infrastructure.</p>
<div class="d-flex align-items-center justify-content-center gap-4 flex-wrap mb-5">
<a href="/blog/2026/03/04/building-the-ai-native-future-of-go-micro-with-claude/"><img src="/images/sponsors/anthropic.svg" alt="Anthropic" class="sponsor-logo" /></a>
<a href="/blog/2026/06/23/go-micro-joins-openai-s-codex-for-open-source/"><img src="/images/sponsors/openai.svg" alt="OpenAI" class="sponsor-logo" /></a>
<a href="/blog/2026/05/28/atlas-cloud-sponsors-go-micro-300-ai-models-one-integration/"><img src="/images/sponsors/atlas-cloud-logo.svg" alt="Atlas Cloud" class="sponsor-logo" /></a>
</div>
<p class="text-light mb-3">Want to support Go Micro and put your logo here? Or running it in production and need a hand?</p>
<div class="d-flex gap-3 justify-content-center">
<a href="https://discord.gg/G8Gk5j3uXr" class="btn btn-outline-go">Become a sponsor</a>
<a href="/support" class="btn btn-go">Commercial support</a>
</div>
{{% /blocks/section %}}

{{% blocks/section color="dark-blue" pattern=true padding="py-5" type="text-center" %}}
<h2>Trusted by Developers</h2>
<p class="text-light mb-4">23,000+ stars on GitHub. Production-ready. Apache 2.0 licensed.</p>
<div class="d-flex gap-3 justify-content-center flex-wrap">
<a href="/docs/getting-started.html" class="btn btn-go">Get Started</a>
<a href="/docs/" class="btn btn-outline-go">Read the Docs</a>
<a href="https://discord.gg/G8Gk5j3uXr" class="btn btn-outline-go">Join Discord</a>
<a href="/blog/" class="btn btn-outline-go">Blog</a>
</div>
{{% /blocks/section %}}

