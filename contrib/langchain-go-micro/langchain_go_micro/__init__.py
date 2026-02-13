"""LangChain Go Micro Integration.

This package provides LangChain integration for Go Micro services through
the Model Context Protocol (MCP).
"""

from langchain_go_micro.toolkit import GoMicroToolkit, GoMicroConfig
from langchain_go_micro.exceptions import GoMicroError, GoMicroConnectionError, GoMicroAuthError

__version__ = "0.1.0"
__all__ = [
    "GoMicroToolkit",
    "GoMicroConfig", 
    "GoMicroError",
    "GoMicroConnectionError",
    "GoMicroAuthError",
]
