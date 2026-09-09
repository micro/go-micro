"""Custom exceptions for LangChain Go Micro integration."""


class GoMicroError(Exception):
    """Base exception for Go Micro integration errors."""
    pass


class GoMicroConnectionError(GoMicroError):
    """Raised when unable to connect to MCP gateway."""
    pass


class GoMicroAuthError(GoMicroError):
    """Raised when authentication fails."""
    pass


class GoMicroToolError(GoMicroError):
    """Raised when tool execution fails."""
    pass
