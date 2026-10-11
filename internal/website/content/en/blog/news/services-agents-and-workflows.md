---
title: "Services, agents, and workflows"
linkTitle: "Services, agents, and workflows"
date: 2026-10-11
description: "Where Go Micro is heading, and a simpler way to start building with it."
category:
- news
tags:
- services
- agents
- workflows
---

Go Micro began with a practical problem: building services in Go meant repeatedly
writing the code to find them, call them, and pass messages between them. The
framework put that work behind interfaces for clients, servers, registries,
brokers, transports, and codecs. Storage and configuration followed.

That remains the foundation. The work on agents builds on it: a service exposes
an operation, and an agent can use that operation as a tool. Workflows let you
put those operations into a defined sequence.

## Start with a service

Consider an application that stores notes. Its service might expose methods to
list, create, and update them. Those methods contain the application logic,
including validation and access checks. Other Go programs call them over RPC.

Adding an agent gives someone another way to use those same operations. They
can ask it to find a note or record something from a conversation. The model
chooses a tool and supplies its arguments; the service still performs the work.

The service remains useful on its own. You can test its handlers, call it from
another service, or put a web interface in front of it. You can change the model
without rewriting the operations it calls.

## Program the agent

An agent needs a model, instructions, and tools. In Go Micro, you configure these
in Go. You can select which services it uses, add function tools, and provide
your own implementations of the framework's interfaces.

The CLI gives you access to what you build. Run your agent, then use
`micro chat` to interact with it. With one registered agent, chat connects to
that agent automatically. Your Go program defines its behaviour.

This is where the framework's value is: developers can build agents around their
own systems and extend the components when the defaults no longer fit.

## Use a workflow for a known sequence

Some work has an order you already know. Fetch records, process them, then save
the result. A workflow expresses those steps directly, and a step can call a
service, ask an agent to do part of the work, or run a Go function.

Use the model where a decision is needed. Keep the sequence in code where it is
already defined. If a step changes data, account for retries: an interrupted
operation may run again.

## A simpler place to start

The documentation had accumulated too many starting points. We've replaced the
main navigation with five pages: Getting started, Services, Agents, Workflows,
and Reference.

The guides follow one small service. First you write and call it. Then you give
an agent access to it. Finally, you call it from a workflow. The first step needs
no model or API key.

[Start with the guide](/docs/quickstart.html), or go straight to
[services](/docs/services.html), [agents](/docs/agents.html), or
[workflows](/docs/workflows.html). The guides currently use the development
version on `master`.

The direction for Go Micro is to make these parts work well together while
keeping them independently useful. The measure is whether a developer can get
something running, understand how it works, and change it to fit their own
application.
