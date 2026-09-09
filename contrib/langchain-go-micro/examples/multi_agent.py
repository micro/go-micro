"""Multi-agent workflow example.

This example demonstrates how to create specialized agents for different
services and coordinate between them.
"""

from langchain_go_micro import GoMicroToolkit
from langchain.agents import initialize_agent, AgentType
from langchain_openai import ChatOpenAI


def main():
    """Run multi-agent example."""
    # Connect to MCP gateway
    toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
    
    # Create LLM
    llm = ChatOpenAI(model="gpt-4", temperature=0)
    
    # Create specialized agents for different services
    print("Creating specialized agents...")
    
    # Agent 1: User management
    user_tools = toolkit.get_tools(service_filter="users")
    user_agent = initialize_agent(
        user_tools,
        llm,
        agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION,
        verbose=True
    )
    print(f"User agent: {len(user_tools)} tools")
    
    # Agent 2: Blog management
    blog_tools = toolkit.get_tools(service_filter="blog")
    blog_agent = initialize_agent(
        blog_tools,
        llm,
        agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION,
        verbose=True
    )
    print(f"Blog agent: {len(blog_tools)} tools")
    
    # Coordinate between agents
    print("\n" + "="*60)
    print("Multi-agent workflow")
    print("="*60)
    
    # Step 1: Create a user
    print("\nStep 1: Creating user...")
    user_result = user_agent.run(
        "Create a user named Bob Smith with email bob@example.com"
    )
    print(f"User created: {user_result}")
    
    # Step 2: Create a blog post for that user
    print("\nStep 2: Creating blog post...")
    blog_result = blog_agent.run(
        f"Create a blog post titled 'Hello World' with content "
        f"'This is my first post' by user {user_result}"
    )
    print(f"Blog post created: {blog_result}")
    
    # Step 3: List user's posts
    print("\nStep 3: Listing user's posts...")
    posts = blog_agent.run(f"List all blog posts by {user_result}")
    print(f"User's posts: {posts}")


if __name__ == "__main__":
    main()
