"""Basic LlamaIndex agent example using Go Micro services.

This example shows how to create a simple LlamaIndex agent that can
interact with Go Micro services through the MCP gateway.
"""

from go_micro_llamaindex import GoMicroToolkit
from llama_index.core.agent import ReActAgent
from llama_index.llms.openai import OpenAI


def main():
    """Run basic agent example."""
    # Initialize toolkit from MCP gateway
    print("Connecting to MCP gateway...")
    toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")

    # Get available tools
    tools = toolkit.get_tools()
    print(f"\nDiscovered {len(tools)} tools:")
    for tool in tools:
        print(f"  - {tool.metadata.name}: {tool.metadata.description}")

    # Create LlamaIndex ReAct agent
    print("\nCreating LlamaIndex agent...")
    llm = OpenAI(model="gpt-4", temperature=0)
    agent = ReActAgent.from_tools(tools, llm=llm, verbose=True)

    # Example queries
    queries = [
        "Create a user named Alice with email alice@example.com",
        "Get the user we just created",
    ]

    for query in queries:
        print(f"\n{'='*60}")
        print(f"Query: {query}")
        print("=" * 60)
        response = agent.chat(query)
        print(f"\nResult: {response}")


if __name__ == "__main__":
    main()
