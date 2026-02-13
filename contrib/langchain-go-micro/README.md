# LangChain Go Micro Integration

[![PyPI version](https://badge.fury.io/py/langchain-go-micro.svg)](https://badge.fury.io/py/langchain-go-micro)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Official LangChain integration for Go Micro services. This package enables LangChain agents to discover and call Go Micro microservices through the Model Context Protocol (MCP).

## Features

- 🔍 **Automatic Service Discovery** - Discovers available services from MCP gateway
- 🛠️ **Dynamic Tool Generation** - Converts service endpoints into LangChain tools
- 📝 **Rich Descriptions** - Uses service metadata for accurate tool descriptions
- 🔐 **Authentication Support** - Bearer token auth with scope-based permissions
- ⚡ **Type-Safe** - Fully typed with Python 3.8+ type hints
- 🎯 **Easy Integration** - Works with any LangChain agent

## Installation

```bash
pip install langchain-go-micro
```

## Quick Start

### 1. Start Your Go Micro Services

```bash
# Start MCP gateway
micro mcp serve --address :3000
```

### 2. Create LangChain Agent

```python
from langchain_go_micro import GoMicroToolkit
from langchain.agents import initialize_agent, AgentType
from langchain_openai import ChatOpenAI

# Initialize toolkit from MCP gateway
toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")

# Create agent
llm = ChatOpenAI(model="gpt-4")
agent = initialize_agent(
    toolkit.get_tools(),
    llm,
    agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION,
    verbose=True
)

# Use the agent!
result = agent.run("Create a user named Alice with email alice@example.com")
print(result)
```

## Usage Examples

### Basic Tool Discovery

```python
from langchain_go_micro import GoMicroToolkit

# Connect to MCP gateway
toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")

# List available tools
for tool in toolkit.get_tools():
    print(f"Tool: {tool.name}")
    print(f"Description: {tool.description}")
    print()
```

### Authentication

```python
from langchain_go_micro import GoMicroToolkit

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
from langchain_go_micro import GoMicroToolkit

toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")

# Get only user service tools
user_tools = toolkit.get_tools(service_filter="users")

# Get tools matching a pattern
blog_tools = toolkit.get_tools(name_pattern="blog.*")
```

### Custom Tool Selection

```python
from langchain_go_micro import GoMicroToolkit

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

### Multi-Agent Workflows

```python
from langchain_go_micro import GoMicroToolkit
from langchain.agents import initialize_agent, AgentType
from langchain_openai import ChatOpenAI

# Create specialized agents for different services
toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")

# Agent 1: User management
user_agent = initialize_agent(
    toolkit.get_tools(service_filter="users"),
    ChatOpenAI(model="gpt-4"),
    agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION
)

# Agent 2: Order processing
order_agent = initialize_agent(
    toolkit.get_tools(service_filter="orders"),
    ChatOpenAI(model="gpt-4"),
    agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION
)

# Coordinate between agents
user = user_agent.run("Create user Alice")
order = order_agent.run(f"Create order for user {user['id']}")
```

### Error Handling

```python
from langchain_go_micro import GoMicroToolkit, GoMicroError

try:
    toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
    tools = toolkit.get_tools()
except GoMicroError as e:
    print(f"Error: {e}")
    # Handle error (gateway unreachable, auth failed, etc.)
```

### Advanced Configuration

```python
from langchain_go_micro import GoMicroToolkit, GoMicroConfig

config = GoMicroConfig(
    gateway_url="http://localhost:3000",
    auth_token="your-token",
    timeout=30,  # Request timeout in seconds
    retry_count=3,  # Number of retries on failure
    retry_delay=1.0,  # Delay between retries
    verify_ssl=True,  # SSL certificate verification
)

toolkit = GoMicroToolkit(config)
tools = toolkit.get_tools()
```

## API Reference

### GoMicroToolkit

Main class for interacting with Go Micro services.

#### Methods

- `from_gateway(gateway_url, auth_token=None, **kwargs)` - Create toolkit from MCP gateway
- `get_tools(service_filter=None, name_pattern=None, include=None, exclude=None)` - Get LangChain tools
- `refresh()` - Refresh tool list from gateway
- `call_tool(tool_name, arguments)` - Call a tool directly

### GoMicroConfig

Configuration for the toolkit.

#### Parameters

- `gateway_url` (str) - MCP gateway URL
- `auth_token` (str, optional) - Bearer authentication token
- `timeout` (int) - Request timeout in seconds (default: 30)
- `retry_count` (int) - Number of retries (default: 3)
- `retry_delay` (float) - Delay between retries in seconds (default: 1.0)
- `verify_ssl` (bool) - Verify SSL certificates (default: True)

## Integration with LangChain Components

### With LangChain Agents

```python
from langchain_go_micro import GoMicroToolkit
from langchain.agents import initialize_agent, AgentType
from langchain_openai import ChatOpenAI

toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
llm = ChatOpenAI(model="gpt-4")

agent = initialize_agent(
    toolkit.get_tools(),
    llm,
    agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION,
    verbose=True
)
```

### With LangChain Memory

```python
from langchain_go_micro import GoMicroToolkit
from langchain.agents import initialize_agent, AgentType
from langchain_openai import ChatOpenAI
from langchain.memory import ConversationBufferMemory

toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
memory = ConversationBufferMemory(memory_key="chat_history")

agent = initialize_agent(
    toolkit.get_tools(),
    ChatOpenAI(model="gpt-4"),
    agent=AgentType.CONVERSATIONAL_REACT_DESCRIPTION,
    memory=memory,
    verbose=True
)
```

### With Custom LLMs

```python
from langchain_go_micro import GoMicroToolkit
from langchain.agents import initialize_agent, AgentType
from langchain_anthropic import ChatAnthropic

toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")

# Use Claude instead of GPT
agent = initialize_agent(
    toolkit.get_tools(),
    ChatAnthropic(model="claude-3-sonnet-20240229"),
    agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION,
    verbose=True
)
```

## Requirements

- Python 3.8+
- LangChain >= 0.1.0
- requests >= 2.31.0

## Development

### Setup

```bash
git clone https://github.com/micro/go-micro
cd go-micro/contrib/langchain-go-micro

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
pytest --cov=langchain_go_micro

# Run specific test
pytest tests/test_toolkit.py
```

### Code Formatting

```bash
# Format code
black langchain_go_micro tests

# Check types
mypy langchain_go_micro

# Lint
ruff check langchain_go_micro
```

## Examples

See the [examples](./examples) directory for complete examples:

- [basic_agent.py](./examples/basic_agent.py) - Simple agent example
- [multi_agent.py](./examples/multi_agent.py) - Multi-agent workflow
- [with_memory.py](./examples/with_memory.py) - Agent with conversation memory
- [custom_llm.py](./examples/custom_llm.py) - Using different LLMs

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
- [LangChain](https://python.langchain.com/)
- [Issue Tracker](https://github.com/micro/go-micro/issues)

## Support

- GitHub Discussions: https://github.com/micro/go-micro/discussions
- Discord: https://discord.gg/jwTYuUVAGh
