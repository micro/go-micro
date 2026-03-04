# Go Micro Roadmap

This roadmap outlines the planned features and improvements for Go Micro. Community feedback and contributions are welcome!

> **See [internal/docs/ROADMAP_2026.md](internal/docs/ROADMAP_2026.md) for the AI-Native Era roadmap** focused on MCP integration, agent-first development, and business sustainability. This document covers general framework improvements.

## Current Focus (Q1 2026) - COMPLETE

### Documentation & Developer Experience
- [x] Modernize documentation structure
- [x] Add learn-by-example guides
- [x] Update issue templates
- [x] MCP integration documentation
- [x] Agent playground and MCP tools registry
- [ ] Create video tutorials
- [ ] Interactive documentation site
- [ ] Plugin discovery dashboard

### AI & Model Integration
- [x] AI package with provider abstraction (`ai.Model` interface)
- [x] Anthropic Claude provider (`ai/anthropic`)
- [x] OpenAI GPT provider (`ai/openai`)
- [x] Tool execution with auto-calling support
- [x] Streaming support via `ai.Stream`

### Observability
- [ ] OpenTelemetry native support
- [ ] Auto-instrumentation for handlers
- [ ] Metrics export standardization
- [ ] Distributed tracing examples
- [ ] Integration with popular observability platforms

### Developer Tools
- [x] `micro run` with hot reload and unified gateway
- [x] `micro deploy` with SSH + systemd deployment
- [x] `micro mcp` command suite (serve, list, test, docs, export)
- [ ] `micro dev` with enhanced hot reload
- [ ] Service templates (`micro new --template`)
- [ ] Better error messages with suggestions
- [ ] Debug tooling improvements
- [ ] VS Code extension for Go Micro

## Q2 2026

### Production Readiness
- [x] Health check standardization
- [x] Graceful shutdown improvements
- [ ] Resource cleanup best practices
- [ ] Load testing framework integration
- [ ] Performance benchmarking suite

### Cloud Native
- [ ] Kubernetes operator
- [ ] Helm charts for common setups
- [ ] Service mesh integration guides (Istio, Linkerd)
- [ ] Cloud provider quickstarts (AWS, GCP, Azure)
- [ ] Multi-cluster patterns

### Security
- [x] Bearer token authentication for MCP
- [x] Per-tool scope enforcement
- [x] Audit logging
- [x] Rate limiting
- [ ] mTLS by default option
- [ ] Secret management integration (Vault, AWS Secrets Manager)
- [ ] RBAC improvements
- [ ] Security audit and hardening
- [ ] CVE scanning and response process

## Q3 2026

### Plugin Ecosystem
- [ ] Plugin marketplace/registry
- [ ] Plugin quality standards
- [ ] Community plugin contributions
- [ ] Plugin compatibility matrix
- [ ] Auto-discovery of available plugins

### Streaming & Async
- [ ] Improved streaming support
- [x] Server-sent events (SSE) support (via MCP gateway)
- [ ] WebSocket plugin
- [ ] Event sourcing patterns
- [ ] CQRS examples

### Testing
- [ ] Mock generation tooling
- [ ] Integration test helpers
- [ ] Contract testing support
- [ ] Chaos engineering examples
- [ ] E2E testing framework

## Q4 2026

### Performance
- [ ] Connection pooling optimizations
- [ ] Zero-allocation paths
- [ ] gRPC performance improvements
- [ ] Caching strategies guide
- [ ] Performance profiling tools

### Developer Productivity
- [ ] Code generation improvements
- [ ] Better IDE support
- [ ] Debugging tools
- [ ] Migration automation tools
- [ ] Upgrade helpers

### Community
- [ ] Regular blog posts and case studies
- [ ] Community spotlight program
- [ ] Contribution rewards
- [ ] Monthly community calls
- [ ] Conference presence

## Long-term Vision

### Core Framework
- Maintain backward compatibility (Go Micro v5+)
- Progressive disclosure of complexity
- Best-in-class developer experience
- Production-grade reliability
- Comprehensive plugin ecosystem

### Ecosystem Goals
- 100+ production deployments documented
- 50+ community plugins
- Active contributor community
- Regular releases (monthly patches, quarterly features)
- Comprehensive benchmarks vs alternatives

### Differentiation
- **Batteries included, fully swappable** - Start simple, scale complex
- **Zero-config local development** - No infrastructure required to start
- **AI-native by default** - Every service is an MCP tool automatically
- **Plugin ecosystem in-repo** - No version compatibility hell
- **Progressive complexity** - Learn as you grow
- **Cloud-native first** - Built for Kubernetes and containers

## Contributing

We welcome contributions to any roadmap items! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### High Priority Areas
1. Documentation improvements (guides, tutorials)
2. Multi-protocol MCP support (WebSocket, gRPC)
3. Agent SDK integrations (LlamaIndex, AutoGPT)
4. OpenTelemetry integration
5. Kubernetes operator and Helm charts

### How to Contribute
- Pick an item from the roadmap
- Open an issue to discuss approach
- Submit a PR with implementation
- Help review others' contributions

## Feedback

Have suggestions for the roadmap?

- Open a [feature request](.github/ISSUE_TEMPLATE/feature_request.md)
- Start a discussion in GitHub Discussions
- Comment on existing roadmap issues

## Version Compatibility

We follow semantic versioning:
- Major versions (v5 → v6): Breaking changes
- Minor versions (v5.3 → v5.4): New features, backward compatible
- Patch versions (v5.3.0 → v5.3.1): Bug fixes, no API changes

## Support Timeline

- v5: Active development (current)
- v4: Security fixes only (until v6 release)
- v3: End of life

---

Last updated: March 2026

This roadmap is subject to change based on community needs and priorities.
