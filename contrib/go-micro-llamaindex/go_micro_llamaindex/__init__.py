"""LlamaIndex Go Micro Integration.

This package provides LlamaIndex integration for Go Micro services through
the Model Context Protocol (MCP).
"""

from go_micro_llamaindex.toolkit import GoMicroToolkit, GoMicroConfig
from go_micro_llamaindex.exceptions import GoMicroError, GoMicroConnectionError, GoMicroAuthError

__version__ = "0.1.0"
__all__ = [
    "GoMicroToolkit",
    "GoMicroConfig",
    "GoMicroError",
    "GoMicroConnectionError",
    "GoMicroAuthError",
]
