---
title: Integrations
description: Connect Go Micro to other services and providers, or share your own integration.
type: docs
menu:
  main:
    weight: 25
params:
  body_class: td-no-left-sidebar
  hide_feedback: true
---

Connect Go Micro to the systems you use. Vendor-specific plugins can live in their own repositories and implement the framework's interfaces.

[Propose an integration](https://github.com/micro/go-micro/issues/new?template=integration.md) · [Submit a listing PR](https://github.com/micro/go-micro/edit/master/internal/website/content/en/integrations.md)

## Built into Go Micro

- [Models](docs/guides/ai-provider-guide.md) — supported providers and configurable endpoints for OpenAI-compatible APIs.
- [MCP](docs/mcp.md) — expose services as tools for other agents.
- [Plugins](docs/plugins.md) — replace registry, broker, transport, storage and other components through Go interfaces.

## Community integrations

Community listings are free. Integrations are maintained by their authors; a listing does not mean Micro maintains or endorses the code.

To add yours, open an issue to discuss it or submit a PR to this page. Include:

- A short description and a link to the source repository.
- The maintainer, license and supported Go Micro versions.
- Installation instructions and a working example.
- Tests covering its use with Go Micro.

Keep vendor-specific code in your own repository. General improvements and bug fixes to Go Micro are welcome as ordinary PRs and do not require sponsorship. Listings are reviewed for relevance, clarity and functionality.

## Sponsored integrations

Sponsorship funds the framework's maintenance and offers featured placement on this page. Featured listings will be clearly marked **Sponsored**, with the maintainer identified. There are no sponsored listings yet.

[Open an integration issue](https://github.com/micro/go-micro/issues/new?template=integration.md) to discuss placement, or [sponsor Go Micro](https://github.com/sponsors/micro). Agree the placement and terms before paying; a general sponsorship does not automatically create a listing.

Sponsorship does not guarantee code acceptance, endorsement or maintenance by Micro. When sponsorship ends, featured placement ends; an eligible community listing can remain.

Integration development, compatibility testing and ongoing maintenance are available as separately scoped paid work. Use the same issue template to tell us what you need.
