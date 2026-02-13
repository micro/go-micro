"""LangChain toolkit for Go Micro services."""

import json
import re
from typing import Any, Dict, List, Optional, Callable
from dataclasses import dataclass

import requests
from langchain.tools import Tool
from pydantic import BaseModel, Field

from langchain_go_micro.exceptions import (
    GoMicroConnectionError,
    GoMicroAuthError,
    GoMicroToolError,
)


@dataclass
class GoMicroConfig:
    """Configuration for Go Micro MCP gateway connection.
    
    Attributes:
        gateway_url: URL of the MCP gateway (e.g., http://localhost:3000)
        auth_token: Optional bearer authentication token
        timeout: Request timeout in seconds
        retry_count: Number of retries on failure
        retry_delay: Delay between retries in seconds
        verify_ssl: Whether to verify SSL certificates
    """
    
    gateway_url: str
    auth_token: Optional[str] = None
    timeout: int = 30
    retry_count: int = 3
    retry_delay: float = 1.0
    verify_ssl: bool = True


class GoMicroTool(BaseModel):
    """Represents a Go Micro service tool.
    
    Attributes:
        name: Tool name (e.g., "users.Users.Get")
        service: Service name (e.g., "users")
        endpoint: Endpoint name (e.g., "Users.Get")
        description: Tool description
        example: Example input JSON
        scopes: Required auth scopes
        metadata: Additional metadata from service
    """
    
    name: str
    service: str
    endpoint: str
    description: str
    example: Optional[str] = None
    scopes: Optional[List[str]] = None
    metadata: Dict[str, str] = Field(default_factory=dict)


class GoMicroToolkit:
    """LangChain toolkit for Go Micro services.
    
    This class provides integration between LangChain and Go Micro services
    via the Model Context Protocol (MCP) gateway.
    
    Example:
        >>> toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
        >>> tools = toolkit.get_tools()
        >>> for tool in tools:
        ...     print(f"Tool: {tool.name}")
    """
    
    def __init__(self, config: GoMicroConfig):
        """Initialize the toolkit.
        
        Args:
            config: Configuration for MCP gateway connection
        """
        self.config = config
        self._tools: Optional[List[GoMicroTool]] = None
        self._session = requests.Session()
        
        # Set up authentication
        if config.auth_token:
            self._session.headers.update({
                "Authorization": f"Bearer {config.auth_token}"
            })
    
    @classmethod
    def from_gateway(
        cls,
        gateway_url: str,
        auth_token: Optional[str] = None,
        **kwargs: Any
    ) -> "GoMicroToolkit":
        """Create toolkit from MCP gateway URL.
        
        Args:
            gateway_url: URL of the MCP gateway
            auth_token: Optional bearer authentication token
            **kwargs: Additional configuration options
            
        Returns:
            GoMicroToolkit instance
            
        Example:
            >>> toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
        """
        config = GoMicroConfig(
            gateway_url=gateway_url,
            auth_token=auth_token,
            **kwargs
        )
        return cls(config)
    
    def _make_request(
        self,
        method: str,
        path: str,
        **kwargs: Any
    ) -> requests.Response:
        """Make HTTP request to MCP gateway.
        
        Args:
            method: HTTP method (GET, POST, etc.)
            path: API path
            **kwargs: Additional request arguments
            
        Returns:
            Response object
            
        Raises:
            GoMicroConnectionError: If connection fails
            GoMicroAuthError: If authentication fails
        """
        url = f"{self.config.gateway_url}{path}"
        kwargs.setdefault("timeout", self.config.timeout)
        kwargs.setdefault("verify", self.config.verify_ssl)
        
        try:
            response = self._session.request(method, url, **kwargs)
            
            if response.status_code == 401:
                raise GoMicroAuthError("Authentication failed")
            elif response.status_code == 403:
                raise GoMicroAuthError("Forbidden: insufficient permissions")
            
            response.raise_for_status()
            return response
            
        except requests.ConnectionError as e:
            raise GoMicroConnectionError(
                f"Failed to connect to MCP gateway at {url}: {e}"
            )
        except requests.Timeout as e:
            raise GoMicroConnectionError(
                f"Request to MCP gateway timed out: {e}"
            )
        except requests.RequestException as e:
            if isinstance(e, (GoMicroConnectionError, GoMicroAuthError)):
                raise
            raise GoMicroConnectionError(f"Request failed: {e}")
    
    def refresh(self) -> None:
        """Refresh tool list from MCP gateway.
        
        Raises:
            GoMicroConnectionError: If unable to connect to gateway
        """
        response = self._make_request("GET", "/mcp/tools")
        data = response.json()
        
        tools_data = data.get("tools", [])
        self._tools = [
            GoMicroTool(
                name=tool["name"],
                service=tool["service"],
                endpoint=tool["endpoint"],
                description=tool.get("description", ""),
                example=tool.get("example"),
                scopes=tool.get("scopes"),
                metadata=tool.get("metadata", {})
            )
            for tool in tools_data
        ]
    
    def get_tools(
        self,
        service_filter: Optional[str] = None,
        name_pattern: Optional[str] = None,
        include: Optional[List[str]] = None,
        exclude: Optional[List[str]] = None,
    ) -> List[Tool]:
        """Get LangChain tools from Go Micro services.
        
        Args:
            service_filter: Filter tools by service name
            name_pattern: Filter tools by name pattern (regex)
            include: List of tool names to include
            exclude: List of tool names to exclude
            
        Returns:
            List of LangChain Tool objects
            
        Example:
            >>> toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
            >>> # Get all tools
            >>> all_tools = toolkit.get_tools()
            >>> # Get only user service tools
            >>> user_tools = toolkit.get_tools(service_filter="users")
            >>> # Get specific tools
            >>> selected_tools = toolkit.get_tools(include=["users.Users.Get"])
        """
        if self._tools is None:
            self.refresh()
        
        tools = self._tools or []
        
        # Apply filters
        if service_filter:
            tools = [t for t in tools if t.service == service_filter]
        
        if name_pattern:
            pattern = re.compile(name_pattern)
            tools = [t for t in tools if pattern.match(t.name)]
        
        if include:
            tools = [t for t in tools if t.name in include]
        
        if exclude:
            tools = [t for t in tools if t.name not in exclude]
        
        # Convert to LangChain tools
        return [self._create_langchain_tool(tool) for tool in tools]
    
    def _create_langchain_tool(self, tool: GoMicroTool) -> Tool:
        """Create a LangChain Tool from a GoMicroTool.
        
        Args:
            tool: GoMicroTool to convert
            
        Returns:
            LangChain Tool object
        """
        def tool_func(arguments: str) -> str:
            """Execute the tool.
            
            Args:
                arguments: JSON string with tool arguments
                
            Returns:
                JSON string with tool result
            """
            return self.call_tool(tool.name, arguments)
        
        # Build description with example if available
        description = tool.description
        if tool.example:
            description += f"\n\nExample input: {tool.example}"
        
        return Tool(
            name=tool.name,
            func=tool_func,
            description=description,
        )
    
    def call_tool(self, tool_name: str, arguments: str) -> str:
        """Call a specific tool directly.
        
        Args:
            tool_name: Name of the tool to call
            arguments: JSON string with tool arguments
            
        Returns:
            JSON string with tool result
            
        Raises:
            GoMicroToolError: If tool execution fails
            
        Example:
            >>> toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
            >>> result = toolkit.call_tool(
            ...     "users.Users.Get",
            ...     '{"id": "user-123"}'
            ... )
        """
        # Parse arguments
        try:
            args = json.loads(arguments) if isinstance(arguments, str) else arguments
        except json.JSONDecodeError as e:
            raise GoMicroToolError(f"Invalid JSON arguments: {e}")
        
        # Make request
        try:
            response = self._make_request(
                "POST",
                "/mcp/call",
                json={"name": tool_name, "arguments": args}
            )
            return json.dumps(response.json())
        except requests.RequestException as e:
            raise GoMicroToolError(f"Tool execution failed: {e}")
    
    def list_tools(self) -> List[GoMicroTool]:
        """Get raw list of available tools.
        
        Returns:
            List of GoMicroTool objects
            
        Example:
            >>> toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
            >>> for tool in toolkit.list_tools():
            ...     print(f"{tool.name}: {tool.description}")
        """
        if self._tools is None:
            self.refresh()
        return self._tools or []
