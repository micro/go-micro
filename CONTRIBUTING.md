# Contributing to Go Micro

Guidelines/instructions for contributing.

## Code of Conduct

Respectful, inclusive, collaborative.

## How Go Micro is built

Go Micro is developed by an **autonomous improvement loop** — a planner, a
generator, and a separate evaluator, running as scheduled GitHub Actions with a
human setting direction. It's the framework's own thesis (an agent operating a
system) pointed at itself: an agent harness, built by agents. The full process —
the planner → generator → evaluator pipeline, the correctness-only merge gate, and
the guardrails — is documented in
[`internal/docs/CONTINUOUS_IMPROVEMENT.md`](internal/docs/CONTINUOUS_IMPROVEMENT.md).
Human contributions follow the same gate: green CI, one concern per PR.

## Getting Started

1. Fork repo
2. Clone: `git clone https://github.com/YOUR_USERNAME/go-micro.git`
3. Add upstream: `git remote add upstream https://github.com/micro/go-micro.git`
4. Feature branch: `git checkout -b feature/my-feature`

## Development Setup

```bash
# Install dependencies
go mod download

# Install development tools
make install-tools

# Run tests
make test

# Run tests with race detector and coverage
make test-coverage

# Run linter
make lint

# Format code
make fmt
```

See `make help` for all commands.

## Making Changes

### Code Guidelines

- Follow Go conventions (`gofmt`, `golint`)
- Clear commit messages
- Tests for new functionality
- Update docs for API changes
- Focused PRs (one feature/fix per PR)

### Commit Messages

Conventional commits format:

```
type(scope): subject

body

footer
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Docs changes
- `test`: Test changes
- `refactor`: Refactoring
- `perf`: Performance
- `chore`: Maintenance

Examples:
```
feat(registry): add kubernetes registry plugin
fix(broker): resolve nats connection leak
docs(examples): add streaming example
```

### Testing

- Unit tests for new code
- Ensure existing tests pass
- Integration tests for plugins
- Test Go 1.20+

```bash
# Run specific package tests
go test ./registry/...

# Run with verbose output
go test -v ./...

# Run specific test
go test -run TestMyFunction ./pkg/...

# Optional: Use richgo for colored output
go install github.com/kyoh86/richgo@latest
richgo test -v ./...
```

### Documentation

- Update markdown in `internal/website/docs/`
- Add examples to `internal/website/docs/examples/`
- Update README.md for major features
- Add godoc comments for exported items

## Pull Request Process

1. **Update branch**
   ```bash
   git fetch upstream
   git rebase upstream/master
   ```

2. **Run tests and lint**
   ```bash
   go test ./...
   golangci-lint run
   ```

3. **Push fork**
   ```bash
   git push origin feature/my-feature
   ```

4. **Create PR**
   - Descriptive title
   - Reference issues
   - Describe changes/why
   - Screenshots for UI
   - Mark draft if WIP

5. **Review**
   - Respond promptly
   - Make requested changes
   - Re-request review

### PR Checklist

- [ ] Tests pass locally
- [ ] Code follows Go conventions
- [ ] Docs updated
- [ ] Clear commit messages
- [ ] Branch up to date with master
- [ ] No merge conflicts

## Adding Plugins

New plugins:

1. In interface directory (e.g., `registry/myplugin/`)
2. Implement interface fully
3. Comprehensive tests
4. Usage examples
5. Document config (env vars, options)
6. Add plugin docs

Example structure:
```
registry/myplugin/
├── myplugin.go          # Main implementation
├── myplugin_test.go     # Tests
├── options.go           # Plugin-specific options
└── README.md            # Usage and configuration
```

## Reporting Issues

Before creating issue:

1. Search existing issues
2. Check docs
3. Try latest version

Bug reports:
- Use bug report template
- Minimal reproduction code
- Specify versions (Go, Go Micro, plugins)
- Relevant logs

## Documentation Contributions

Welcome. Ways to edit docs:

- **Fix/edit existing page**: submit PR.
- **Add new page**: create Markdown under `internal/website/content/en/docs/` and open PR.

Site built with Hugo, content in `internal/website/content/en`. 
Local dev instructions (Hugo Extended, `npm ci`, `npm run serve`, production build): [internal/website README](internal/website/README.md).

## Community

- GitHub Issues: Bug reports, feature requests
- GitHub Discussions: Questions, ideas, chat
- Sponsorship: [GitHub Sponsors](https://github.com/sponsors/micro)

## Release Process

Maintainers handle releases:

1. Update CHANGELOG.md
2. Tag: `git tag -a v5.x.x -m "Release v5.x.x"`
3. Push tag: `git push origin v5.x.x`
4. GitHub Actions creates release

## Questions?

- Check [documentation](internal/website/content/en/docs/)
- Browse [examples](internal/website/content/en/docs/examples/)
- Open [question issue](.github/ISSUE_TEMPLATE/question.md)

Thanks for contributing! 🎉