# KnowledgeHub Sandbox

A Docker-based development sandbox with AI coding tools.

## Tools Included

- **opencode** (opencode.ai) - AI coding assistant
- **pi** (pi.dev) - AI pair programmer
- **bun** - JavaScript runtime (included in base image)

## Quick Start

1. Run the sandbox:

```bash
./sandbox-opencode          # runs opencode
./sandbox-pi              # runs pi
```

## Credentials

Credentials are automatically mounted from your host machine:

- **opencode**: `~/.config/opencode` and `~/.local/share/opencode`
- **pi**: `~/.pi`

Your existing credentials will be available in the container.

## How It Works

- The container installs tools (opencode, pi) on first run via npm/bun
- Your host credentials are mounted into the container
- The workspace directory (`./workspace`) is mounted from the host

## Notes

- First run will install dependencies (~2-3 minutes)
- Subsequent runs are faster as tools are cached in the container
