"""Basic LangChain agent example using Go Micro services.

This example shows how to create a simple LangChain agent that can
interact with Go Micro services through the MCP gateway.
"""

from langchain_go_micro import GoMicroToolkit
from langchain.agents import initialize_agent, AgentType
from langchain_openai import ChatOpenAI


def main():
    """Run basic agent example."""
    # Initialize toolkit from MCP gateway
    print("Connecting to MCP gateway...")
    toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
    
    # Get available tools
    tools = toolkit.get_tools()
    print(f"\nDiscovered {len(tools)} tools:")
    for tool in tools:
        print(f"  - {tool.name}: {tool.description}")
    
    # Create LangChain agent
    print("\nCreating LangChain agent...")
    llm = ChatOpenAI(model="gpt-4", temperature=0)
    agent = initialize_agent(
        tools,
        llm,
        agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION,
        verbose=True
    )
    
    # Example queries
    queries = [
        "Create a user named Alice with email alice@example.com",
        "Get the user we just created",
    ]
    
    for query in queries:
        print(f"\n{'='*60}")
        print(f"Query: {query}")
        print('='*60)
        result = agent.run(query)
        print(f"\nResult: {result}")


if __name__ == "__main__":
    main()
