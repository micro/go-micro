# LlamaIndex Go Micro Integration

[![PyPI version](https://badge.fury.io/py/go-micro-llamaindex.svg)](https://badge.fury.io/py/go-micro-llamaindex)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Official LlamaIndex integration for Go Micro services. This package enables LlamaIndex agents to discover and call Go Micro microservices through the Model Context Protocol (MCP).

## Features

- **Automatic Service Discovery** - Discovers available services from MCP gateway
- **Dynamic Tool Generation** - Converts service endpoints into LlamaIndex tools
- **Rich Descriptions** - Uses service metadata for accurate tool descriptions
- **Authentication Support** - Bearer token auth with scope-based permissions
- **RAG Integration** - Combine service tools with LlamaIndex's RAG capabilities
- **Type-Safe** - Fully typed with Python 3.8+ type hints

## Installation

```bash
pip install go-micro-llamaindex
```

## Quick Start

### 1. Start Your Go Micro Services

```bash
# Start MCP gateway
micro mcp serve --address :3000
```

### 2. Create LlamaIndex Agent

```python
from go_micro_llamaindex import GoMicroToolkit
from llama_index.core.agent import ReActAgent
from llama_index.llms.openai import OpenAI

# Initialize toolkit from MCP gateway
toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")

# Create agent
llm = OpenAI(model="gpt-4")
agent = ReActAgent.from_tools(toolkit.get_tools(), llm=llm, verbose=True)

# Use the agent!
response = agent.chat("Create a user named Alice with email alice@example.com")
print(response)
```

## Usage Examples

### Basic Tool Discovery

```python
from go_micro_llamaindex import GoMicroToolkit

# Connect to MCP gateway
toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")

# List available tools
for tool in toolkit.get_tools():
    print(f"Tool: {tool.metadata.name}")
    print(f"Description: {tool.metadata.description}")
    print()
```

### Authentication

```python
from go_micro_llamaindex import GoMicroToolkit

# Create toolkit with authentication
toolkit = GoMicroToolkit.from_gateway(
    gateway_url="http://localhost:3000",
    auth_token="your-bearer-token"
)

# Tools will automatically use the auth token
tools = toolkit.get_tools()
```

### Filter Tools by Service

```python
from go_micro_llamaindex import GoMicroToolkit

toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")

# Get only user service tools
user_tools = toolkit.get_tools(service_filter="users")

# Get tools matching a pattern
blog_tools = toolkit.get_tools(name_pattern="blog.*")
```

### Custom Tool Selection

```python
from go_micro_llamaindex import GoMicroToolkit

toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")

# Select specific tools
selected_tools = toolkit.get_tools(
    include=["users.Users.Get", "users.Users.Create"]
)

# Exclude certain tools
filtered_tools = toolkit.get_tools(
    exclude=["users.Users.Delete"]
)
```

### RAG + Microservices

```python
from go_micro_llamaindex import GoMicroToolkit
from llama_index.core import VectorStoreIndex, Document
from llama_index.core.agent import ReActAgent
from llama_index.core.tools import QueryEngineTool, ToolMetadata
from llama_index.llms.openai import OpenAI

toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")

# Combine service tools with a RAG query engine
index = VectorStoreIndex.from_documents([...])
rag_tool = QueryEngineTool(
    query_engine=index.as_query_engine(),
    metadata=ToolMetadata(name="docs", description="Search documentation"),
)

all_tools = [rag_tool] + toolkit.get_tools()
agent = ReActAgent.from_tools(all_tools, llm=OpenAI(model="gpt-4"))
```

### Multi-Agent Workflows

```python
from go_micro_llamaindex import GoMicroToolkit
from llama_index.core.agent import ReActAgent
from llama_index.llms.openai import OpenAI

toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
llm = OpenAI(model="gpt-4")

# Agent 1: User management
user_agent = ReActAgent.from_tools(
    toolkit.get_tools(service_filter="users"), llm=llm
)

# Agent 2: Blog management
blog_agent = ReActAgent.from_tools(
    toolkit.get_tools(service_filter="blog"), llm=llm
)

# Coordinate between agents
user_result = user_agent.chat("Create user Alice")
blog_result = blog_agent.chat(f"Create blog post for {user_result}")
```

### Error Handling

```python
from go_micro_llamaindex import GoMicroToolkit, GoMicroError

try:
    toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
    tools = toolkit.get_tools()
except GoMicroError as e:
    print(f"Error: {e}")
```

### Advanced Configuration

```python
from go_micro_llamaindex import GoMicroToolkit, GoMicroConfig

config = GoMicroConfig(
    gateway_url="http://localhost:3000",
    auth_token="your-token",
    timeout=30,
    retry_count=3,
    retry_delay=1.0,
    verify_ssl=True,
)

toolkit = GoMicroToolkit(config)
tools = toolkit.get_tools()
```

## API Reference

### GoMicroToolkit

Main class for interacting with Go Micro services.

#### Methods

- `from_gateway(gateway_url, auth_token=None, **kwargs)` - Create toolkit from MCP gateway
- `get_tools(service_filter=None, name_pattern=None, include=None, exclude=None)` - Get LlamaIndex tools
- `refresh()` - Refresh tool list from gateway
- `call_tool(tool_name, arguments)` - Call a tool directly
- `list_tools()` - Get raw list of available tools

### GoMicroConfig

Configuration for the toolkit.

#### Parameters

- `gateway_url` (str) - MCP gateway URL
- `auth_token` (str, optional) - Bearer authentication token
- `timeout` (int) - Request timeout in seconds (default: 30)
- `retry_count` (int) - Number of retries (default: 3)
- `retry_delay` (float) - Delay between retries in seconds (default: 1.0)
- `verify_ssl` (bool) - Verify SSL certificates (default: True)

## Requirements

- Python 3.8+
- llama-index-core >= 0.10.0
- requests >= 2.31.0
- pydantic >= 2.0.0

## Development

### Setup

```bash
git clone https://github.com/micro/go-micro
cd go-micro/contrib/go-micro-llamaindex

# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install in development mode
pip install -e ".[dev]"
```

### Running Tests

```bash
# Run all tests
pytest

# Run with coverage
pytest --cov=go_micro_llamaindex

# Run specific test
pytest tests/test_toolkit.py
```

### Code Formatting

```bash
# Format code
black go_micro_llamaindex tests

# Check types
mypy go_micro_llamaindex

# Lint
ruff check go_micro_llamaindex
```

## Examples

See the [examples](./examples) directory for complete examples:

- [basic_agent.py](./examples/basic_agent.py) - Simple ReAct agent
- [rag_with_services.py](./examples/rag_with_services.py) - RAG combined with microservices

## Troubleshooting

### Gateway Connection Issues

If you can't connect to the MCP gateway:

1. Verify the gateway is running:
```bash
curl http://localhost:3000/health
```

2. Check the gateway URL is correct
3. Verify firewall settings

### Authentication Errors

If you get authentication errors:

1. Verify your token is valid
2. Check the token has required scopes
3. Review gateway logs for details

### Tool Discovery Issues

If tools aren't being discovered:

1. List services from gateway:
```bash
curl http://localhost:3000/mcp/tools
```

2. Verify services are registered
3. Check service metadata is correct

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](../../CONTRIBUTING.md) for details.

## License

Apache 2.0 - See [LICENSE](../../LICENSE) for details.

## Links

- [Go Micro](https://github.com/micro/go-micro)
- [MCP Documentation](../../gateway/mcp/DOCUMENTATION.md)
- [LlamaIndex](https://docs.llamaindex.ai/)
- [Issue Tracker](https://github.com/micro/go-micro/issues)

## Support

- GitHub Discussions: https://github.com/micro/go-micro/discussions
- Discord: https://discord.gg/jwTYuUVAGh
