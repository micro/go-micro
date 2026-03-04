"""RAG with Go Micro services example.

This example demonstrates how to combine LlamaIndex's RAG capabilities
with Go Micro service tools, allowing an agent to both query documents
and interact with microservices.
"""

from go_micro_llamaindex import GoMicroToolkit
from llama_index.core import VectorStoreIndex, Document
from llama_index.core.agent import ReActAgent
from llama_index.core.tools import QueryEngineTool, ToolMetadata
from llama_index.llms.openai import OpenAI


def main():
    """Run RAG + services example."""
    # Initialize toolkit from MCP gateway
    print("Connecting to MCP gateway...")
    toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")

    # Get service tools (e.g., user management)
    service_tools = toolkit.get_tools(service_filter="users")
    print(f"Discovered {len(service_tools)} user service tools")

    # Create a simple document index for RAG
    documents = [
        Document(text="Alice is the admin user with ID user-001."),
        Document(text="Bob is a regular user with ID user-002."),
        Document(text="The blog service supports creating, reading, and deleting posts."),
        Document(text="Users need the 'blog:write' scope to create blog posts."),
    ]

    print("Building document index...")
    index = VectorStoreIndex.from_documents(documents)
    query_engine = index.as_query_engine()

    # Create a query engine tool for RAG
    rag_tool = QueryEngineTool(
        query_engine=query_engine,
        metadata=ToolMetadata(
            name="knowledge_base",
            description="Search the knowledge base for information about users, "
            "services, and permissions. Use this to look up user IDs, "
            "service capabilities, and required scopes.",
        ),
    )

    # Combine RAG tool with service tools
    all_tools = [rag_tool] + service_tools

    # Create agent with both capabilities
    print("\nCreating agent with RAG + service tools...")
    llm = OpenAI(model="gpt-4", temperature=0)
    agent = ReActAgent.from_tools(all_tools, llm=llm, verbose=True)

    # Example: Agent uses RAG to find user ID, then calls service
    queries = [
        "What is Alice's user ID?",
        "Look up Alice's user ID from the knowledge base, then get her full profile from the user service",
        "What scope do I need to create blog posts?",
    ]

    for query in queries:
        print(f"\n{'='*60}")
        print(f"Query: {query}")
        print("=" * 60)
        response = agent.chat(query)
        print(f"\nResult: {response}")


if __name__ == "__main__":
    main()
