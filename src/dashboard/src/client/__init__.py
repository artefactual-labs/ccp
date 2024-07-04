"""
Handles communication with the MCPServer.
"""

import threading

from client.base import Client
from client.ccp import CCPClient
from client.mcp import MCPClient
from django.utils.translation import get_language

client_local = threading.local()


def get_client(user, client_class=None):
    """Return the client for communicating with the MCPServer."""
    if not getattr(client_local, "client", None):
        client_class = client_class or CCPClient
        lang = get_language() or "en"
        client_local.client = client_class(user, lang)

    return client_local.client


__all__ = ("MCPClient", "CCPClient", "Client", "get_client")
