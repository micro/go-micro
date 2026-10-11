---
title: "Pull request #5000"
linkTitle: "Pull request #5000"
date: 2026-10-11T15:46:00+01:00
description: "Go Micro reaches PR #5000. A moment to thank the people behind the project and look at what comes next."
category:
- news
tags:
- community
- go-micro
---

We've just merged [pull request #5000](https://github.com/micro/go-micro/pull/5000)
in Go Micro.

It's a good moment to pause. An open source framework is built through years of
people using it, finding problems, asking questions, contributing fixes, and
challenging decisions. Thank you to everyone who has done that for Go Micro.

## What brought us here

Go Micro started with the building blocks for services in Go: a client, a server,
a registry, a broker, a transport, and a codec. The idea was straightforward.
Write your service and let the framework handle how it communicates. Give each
component an interface and a default implementation, so developers can replace
the parts they need to.

Storage and configuration extended that foundation. More recently, agents have
given services another kind of caller. A service operation can become a tool
that a model uses. Workflows coordinate those operations when the sequence is
known.

The value still comes from something quite ordinary: being able to build useful
software without wiring everything together yourself.

## What #5000 changed

The milestone PR is about consistency. It aligns installation instructions,
makes provider configuration more predictable, and checks the path from a
service to an agent to a workflow.

It also removes a surprising behaviour: an unknown model provider could fall
back to an arbitrary registered provider. Now it fails clearly. Changing an
agent's provider refreshes credentials taken from the environment, and existing
provider-specific environment variables remain supported.

The verification includes a real local RPC call, an agent calling that service
through provider adapters, and a workflow recovering from saved progress without
repeating a completed step. The provider replies in that test are simulated;
the service calls and stored checkpoints are real.

These are small details, but they determine whether the framework does what a
developer expects.

## The next part

Our focus is simplicity. A shorter README. Fewer paths through the documentation.
A clear way to build a service, let an agent use it, and coordinate work.

The same standard applies to the code: understandable interfaces, useful
defaults, and room to extend things in Go. Supporting more models should fit
that design without making every application more complicated.

If you've used Go Micro for years, thank you for sticking with it. If you're
coming to it now, [start with the guides](/docs/quickstart.html). And if something
is harder than it should be, [tell us](https://github.com/micro/go-micro/issues).

That's the work ahead: make the framework easier to understand, easier to use,
and dependable enough to build on.
