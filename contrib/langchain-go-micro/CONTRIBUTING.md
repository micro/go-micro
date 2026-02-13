# Contributing to LangChain Go Micro

Thank you for your interest in contributing to the LangChain Go Micro integration!

## Development Setup

1. Clone the repository:
```bash
git clone https://github.com/micro/go-micro
cd go-micro/contrib/langchain-go-micro
```

2. Create a virtual environment:
```bash
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
```

3. Install in development mode:
```bash
pip install -e ".[dev]"
```

## Running Tests

Run all tests:
```bash
pytest
```

Run with coverage:
```bash
pytest --cov=langchain_go_micro --cov-report=html
```

Run specific tests:
```bash
pytest tests/test_toolkit.py::TestGoMicroToolkit::test_get_tools
```

## Code Style

We use several tools to maintain code quality:

### Black (code formatting)
```bash
black langchain_go_micro tests examples
```

### MyPy (type checking)
```bash
mypy langchain_go_micro
```

### Ruff (linting)
```bash
ruff check langchain_go_micro tests
```

Run all checks:
```bash
black langchain_go_micro tests examples && \
mypy langchain_go_micro && \
ruff check langchain_go_micro tests
```

## Testing with Real Services

To test with real Go Micro services:

1. Start example services:
```bash
cd ../../examples/mcp/documented
go run main.go
```

2. Run integration tests:
```bash
cd contrib/langchain-go-micro
pytest tests/integration/ -v
```

## Submitting Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run tests and code quality checks
5. Commit your changes (`git commit -am 'Add new feature'`)
6. Push to your fork (`git push origin feature/my-feature`)
7. Create a Pull Request

## Pull Request Guidelines

- Include tests for new features
- Update documentation as needed
- Follow existing code style
- Add entry to CHANGELOG.md
- Ensure all tests pass
- Keep changes focused and atomic

## Questions?

- GitHub Discussions: https://github.com/micro/go-micro/discussions
- Discord: https://discord.gg/jwTYuUVAGh
