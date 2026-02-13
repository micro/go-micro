"""Tests for GoMicroToolkit."""

import json
from unittest.mock import Mock, patch

import pytest
import requests

from langchain_go_micro import GoMicroToolkit, GoMicroConfig
from langchain_go_micro.exceptions import (
    GoMicroConnectionError,
    GoMicroAuthError,
)


@pytest.fixture
def mock_gateway_response():
    """Mock MCP gateway response."""
    return {
        "tools": [
            {
                "name": "users.Users.Get",
                "service": "users",
                "endpoint": "Users.Get",
                "description": "Get a user by ID",
                "example": '{"id": "user-123"}',
                "scopes": ["users:read"],
                "metadata": {
                    "description": "Get a user by ID",
                    "example": '{"id": "user-123"}',
                    "scopes": "users:read"
                }
            },
            {
                "name": "users.Users.Create",
                "service": "users",
                "endpoint": "Users.Create",
                "description": "Create a new user",
                "example": '{"name": "Alice", "email": "alice@example.com"}',
                "scopes": ["users:write"],
                "metadata": {}
            }
        ],
        "count": 2
    }


class TestGoMicroConfig:
    """Tests for GoMicroConfig."""
    
    def test_config_defaults(self):
        """Test config default values."""
        config = GoMicroConfig(gateway_url="http://localhost:3000")
        
        assert config.gateway_url == "http://localhost:3000"
        assert config.auth_token is None
        assert config.timeout == 30
        assert config.retry_count == 3
        assert config.retry_delay == 1.0
        assert config.verify_ssl is True
    
    def test_config_custom_values(self):
        """Test config with custom values."""
        config = GoMicroConfig(
            gateway_url="http://localhost:8080",
            auth_token="test-token",
            timeout=60,
            retry_count=5,
            retry_delay=2.0,
            verify_ssl=False
        )
        
        assert config.gateway_url == "http://localhost:8080"
        assert config.auth_token == "test-token"
        assert config.timeout == 60
        assert config.retry_count == 5
        assert config.retry_delay == 2.0
        assert config.verify_ssl is False


class TestGoMicroToolkit:
    """Tests for GoMicroToolkit."""
    
    def test_from_gateway(self):
        """Test creating toolkit from gateway URL."""
        toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
        
        assert toolkit.config.gateway_url == "http://localhost:3000"
        assert toolkit.config.auth_token is None
    
    def test_from_gateway_with_auth(self):
        """Test creating toolkit with authentication."""
        toolkit = GoMicroToolkit.from_gateway(
            "http://localhost:3000",
            auth_token="test-token"
        )
        
        assert toolkit.config.auth_token == "test-token"
        assert "Authorization" in toolkit._session.headers
        assert toolkit._session.headers["Authorization"] == "Bearer test-token"
    
    @patch("requests.Session.request")
    def test_refresh(self, mock_request, mock_gateway_response):
        """Test refreshing tool list."""
        mock_response = Mock()
        mock_response.json.return_value = mock_gateway_response
        mock_response.status_code = 200
        mock_request.return_value = mock_response
        
        toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
        toolkit.refresh()
        
        assert len(toolkit._tools) == 2
        assert toolkit._tools[0].name == "users.Users.Get"
        assert toolkit._tools[1].name == "users.Users.Create"
    
    @patch("requests.Session.request")
    def test_get_tools(self, mock_request, mock_gateway_response):
        """Test getting LangChain tools."""
        mock_response = Mock()
        mock_response.json.return_value = mock_gateway_response
        mock_response.status_code = 200
        mock_request.return_value = mock_response
        
        toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
        tools = toolkit.get_tools()
        
        assert len(tools) == 2
        assert tools[0].name == "users.Users.Get"
        assert tools[1].name == "users.Users.Create"
    
    @patch("requests.Session.request")
    def test_get_tools_with_service_filter(self, mock_request, mock_gateway_response):
        """Test filtering tools by service."""
        mock_response = Mock()
        mock_response.json.return_value = mock_gateway_response
        mock_response.status_code = 200
        mock_request.return_value = mock_response
        
        toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
        tools = toolkit.get_tools(service_filter="users")
        
        assert len(tools) == 2
        for tool in tools:
            assert "users" in tool.name
    
    @patch("requests.Session.request")
    def test_get_tools_with_include(self, mock_request, mock_gateway_response):
        """Test including specific tools."""
        mock_response = Mock()
        mock_response.json.return_value = mock_gateway_response
        mock_response.status_code = 200
        mock_request.return_value = mock_response
        
        toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
        tools = toolkit.get_tools(include=["users.Users.Get"])
        
        assert len(tools) == 1
        assert tools[0].name == "users.Users.Get"
    
    @patch("requests.Session.request")
    def test_get_tools_with_exclude(self, mock_request, mock_gateway_response):
        """Test excluding specific tools."""
        mock_response = Mock()
        mock_response.json.return_value = mock_gateway_response
        mock_response.status_code = 200
        mock_request.return_value = mock_response
        
        toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
        tools = toolkit.get_tools(exclude=["users.Users.Create"])
        
        assert len(tools) == 1
        assert tools[0].name == "users.Users.Get"
    
    @patch("requests.Session.request")
    def test_call_tool(self, mock_request):
        """Test calling a tool directly."""
        mock_response = Mock()
        mock_response.json.return_value = {"user": {"id": "user-123", "name": "Alice"}}
        mock_response.status_code = 200
        mock_request.return_value = mock_response
        
        toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
        result = toolkit.call_tool("users.Users.Get", '{"id": "user-123"}')
        
        result_data = json.loads(result)
        assert result_data["user"]["id"] == "user-123"
    
    @patch("requests.Session.request")
    def test_connection_error(self, mock_request):
        """Test handling connection errors."""
        mock_request.side_effect = requests.ConnectionError("Connection failed")
        
        toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
        
        with pytest.raises(GoMicroConnectionError):
            toolkit.refresh()
    
    @patch("requests.Session.request")
    def test_auth_error(self, mock_request):
        """Test handling authentication errors."""
        mock_response = Mock()
        mock_response.status_code = 401
        mock_request.return_value = mock_response
        
        toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
        
        with pytest.raises(GoMicroAuthError):
            toolkit.refresh()
    
    @patch("requests.Session.request")
    def test_timeout(self, mock_request):
        """Test handling timeouts."""
        mock_request.side_effect = requests.Timeout("Request timed out")
        
        toolkit = GoMicroToolkit.from_gateway("http://localhost:3000")
        
        with pytest.raises(GoMicroConnectionError):
            toolkit.refresh()
