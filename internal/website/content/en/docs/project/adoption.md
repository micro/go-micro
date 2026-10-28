---
title: "Adoption and Sustainability"
---

# Adoption and sustainability

Go Micro does not collect mandatory runtime telemetry. Application prompts,
payloads, service names, source paths, credentials, agent conversations, and
business data must never be collected for project analytics.

## Signals

The project uses privacy-preserving, aggregate signals that already exist:

- release and container download counts;
- public Go module usage data when available;
- aggregate documentation traffic;
- issue and support templates that request exact Go Micro and Go versions; and
- periodic, voluntary user surveys covering deployed version, production or
  evaluation use, integrations, and upgrade blockers.

Any future CLI reporting must be proposed separately, be disabled by default,
be disclosed before use, document deletion and disable controls, and collect
only coarse version/platform/command-category data. It must not be a condition
of using the framework.

## Quarterly review

Maintainers should publish an aggregate review when enough data exists to avoid
identifying respondents. It should cover:

- active major/minor versions and blocked upgrades;
- commonly reported registries, brokers, stores, transports, and model providers;
- production versus evaluation responses and demonstrated roadmap demand;
- sponsor retention and commercial-support inquiries;
- maintainer capacity, issue response time, and release frequency; and
- roadmap, maintenance, and migration work completed.

Sparse data should be reported as unknown rather than inferred. Survey answers
are voluntary, and maintainers should not publish small cohorts that could
identify an organization.

Use the repository's **Production user survey** issue template to collect the
voluntary inputs and the **Quarterly adoption review** template to publish a
report. A review period should be named as `YYYY-QN`; unanswered fields remain
`Unknown`, and free-form responses must be aggregated rather than reproduced.
The first period is 2026 Q4: maintainers track the
[voluntary survey](https://github.com/micro/go-micro/issues/5045) and
[aggregate review](https://github.com/micro/go-micro/issues/5046) in public.
