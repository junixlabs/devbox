#!/usr/bin/env python3
"""
Example: AI agent using devbox MCP server.

Connects to `devbox mcp serve` via stdio, registers as an agent,
executes commands in an isolated workspace, and cleans up on exit.

Usage:
    python agent-script.py --name coder --repo https://github.com/org/repo.git

Requirements: Python 3.8+ (stdlib only, no external dependencies)
"""

import argparse
import json
import subprocess
import sys


class DevboxMCPClient:
    """Minimal MCP client that talks to devbox over stdio."""

    def __init__(self):
        self._id = 0
        self._process = subprocess.Popen(
            ["devbox", "mcp", "serve"],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self._initialize()

    def _next_id(self):
        self._id += 1
        return self._id

    def _send(self, method, params=None):
        """Send a JSON-RPC request and return the result."""
        request = {
            "jsonrpc": "2.0",
            "id": self._next_id(),
            "method": method,
        }
        if params is not None:
            request["params"] = params

        data = json.dumps(request) + "\n"
        self._process.stdin.write(data.encode())
        self._process.stdin.flush()

        line = self._process.stdout.readline()
        if not line:
            raise ConnectionError("MCP server closed connection")

        response = json.loads(line)
        if "error" in response:
            err = response["error"]
            raise RuntimeError(f"MCP error {err.get('code')}: {err.get('message')}")
        return response.get("result")

    def _initialize(self):
        """MCP handshake."""
        self._send("initialize", {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "devbox-agent-script", "version": "1.0.0"},
        })
        self._send("notifications/initialized")

    def call_tool(self, name, arguments=None):
        """Call an MCP tool and return its content."""
        result = self._send("tools/call", {
            "name": name,
            "arguments": arguments or {},
        })
        if result and "content" in result:
            for item in result["content"]:
                if item.get("type") == "text":
                    return json.loads(item["text"])
        return result

    def close(self):
        """Disconnect — triggers agent cleanup on the server side."""
        if self._process.stdin:
            self._process.stdin.close()
        self._process.wait(timeout=10)


def main():
    parser = argparse.ArgumentParser(description="AI agent using devbox MCP")
    parser.add_argument("--name", required=True, help="Agent name")
    parser.add_argument("--repo", help="Git repository to clone")
    parser.add_argument("--cpus", type=float, default=1.0, help="CPU limit (default: 1.0)")
    parser.add_argument("--memory", default="2g", help="Memory limit (default: 2g)")
    args = parser.parse_args()

    client = DevboxMCPClient()
    try:
        # Register agent — auto-creates isolated workspace
        print(f"Registering agent '{args.name}'...")
        agent = client.call_tool("devbox_agent_register", {
            "name": args.name,
            "cpus": args.cpus,
            "memory": args.memory,
        })
        agent_id = agent["agent_id"]
        workspace = agent["workspace"]
        print(f"  Agent ID:  {agent_id}")
        print(f"  Workspace: {workspace}")
        print(f"  Server:    {agent['server']}")

        # Clone repo if provided
        if args.repo:
            print(f"\nCloning {args.repo}...")
            result = client.call_tool("devbox_workspace_exec", {
                "name": workspace,
                "command": f"cd /workspaces && git clone {args.repo}",
            })
            if result["exit_code"] != 0:
                print(f"  Clone failed: {result['stderr']}", file=sys.stderr)
                return 1
            print("  Done.")

        # Example: list files in workspace
        print("\nWorkspace contents:")
        result = client.call_tool("devbox_workspace_exec", {
            "name": workspace,
            "command": "ls -la /workspaces/",
        })
        print(result["stdout"])

        # Example: check resource usage
        print("Resource usage:")
        metrics = client.call_tool("devbox_metrics", {
            "workspace": workspace,
        })
        print(f"  CPU:    {metrics.get('cpu_percent', 0):.1f}%")
        print(f"  Memory: {metrics.get('mem_usage', 0) / 1024 / 1024:.0f}MB")

        # --- Add your agent logic here ---
        # result = client.call_tool("devbox_workspace_exec", {
        #     "name": workspace,
        #     "command": "cd /workspaces/repo && go test ./...",
        # })

        print("\nAgent work complete.")
        return 0

    finally:
        # Disconnect — workspace is auto-destroyed
        print("Disconnecting (workspace will be cleaned up)...")
        client.close()


if __name__ == "__main__":
    sys.exit(main() or 0)
