---
title: "Pull request #5000"
linkTitle: "Pull request #5000"
date: 2026-10-11T15:46:00+01:00
description: "A decade of Go Micro, the move from services to agents and workflows, and a renewed ability to keep building."
category:
- news
tags:
- community
- go-micro
---

We've just merged [pull request #5000](https://github.com/micro/go-micro/pull/5000)
in Go Micro. It feels like a good moment to reflect on a project that has been
part of my life for more than a decade.

Maintaining any piece of software for that long is difficult. The industry
moves, people's needs change, and so does your own interest and energy. Keeping
something useful means revisiting decisions, supporting what people already
depend on, and finding a reason to keep going.

Thank you to everyone who has used Go Micro, contributed to it, reported a
problem, or questioned its direction over the years. That continued involvement
is part of what makes it possible to maintain a project this long.

## Services are still the foundation

Go Micro started with a simple idea: make it easier to build services in Go.
A client, a server, a registry, a broker, a transport, and a codec gave developers
the building blocks for service communication. Storage and configuration grew
around them. Interfaces and useful defaults let people start small and replace
the parts they needed to.

Those services are still the core building blocks of everything we're doing.
They contain the operations, data, and application logic that make a system
useful.

An agent needs a way to act. Services give it that: operations it can discover
and call as tools. A workflow gives those operations a sequence when the order
of work is known. Both build on the same capabilities that another Go program,
an API, or a user interface can use.

That makes the move from services to agents and workflows a natural evolution
of the framework. The service you build remains useful as the ways of
interacting with it change. An agent can choose what to call, while the service
continues to define what that operation does.

## A renewed ability to build

AI has also changed how I work on Go Micro itself.

Tools such as GitHub Copilot, Claude Code, and Codex have brought a renewal of
interest and ability. I can explore an idea, work through changes, and revisit
parts of the project that would previously have taken too much time to tackle.
There is less friction between seeing what needs to change and being able to
do something about it.

Not getting bogged down in the code has helped a lot. It leaves more room to
think about the framework as a whole: what it is for, how the pieces fit
together, and what it feels like for someone trying to use it.

The responsibility for those decisions remains. Generated code still needs
review, testing, and a clear reason to exist. Being able to make more changes
makes it even more important to decide which changes are worth making.

I see that renewed interest beyond this project too. Across the industry,
people are reconsidering what they can build and what existing software can
become. For a long-running project, that is an opportunity to return to its
foundations and find new uses for them.

## Keeping it simple

The next stage of Go Micro is about making services, agents, and workflows fit
together in a way that is easy to understand and extend.

That means keeping services useful on their own, making agents programmable in
Go, and giving developers a clear way to coordinate work. It also means shorter
documentation, fewer competing starting points, and paying attention to the
whole experience rather than just adding capabilities.

After a decade, there is still more to build. AI has helped restore the energy
and ability to do that work. The aim remains familiar: make it easier for
developers to build useful software.

[Start with the guides](/docs/quickstart.html), or
[join us on GitHub](https://github.com/micro/go-micro).
