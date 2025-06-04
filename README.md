# SimCon - Simple Container Runtime

SimCon is a minimal container runtime implementation written in Go. It provides basic container isolation features similar to Docker but with a simpler implementation.

## Features

- Container process isolation using Linux namespaces
- Resource limits using cgroups
- Basic filesystem isolation
- Simple networking support

## Requirements

- Linux operating system
- Go 1.21 or later
- Root privileges (for container operations)

## Installation

```bash
go install github.com/yoonhyunwoo/simcon/cmd/simcon@latest
```

## Usage

```bash
# Run a container
simcon run <image> <command>

# List running containers
simcon ps

# Stop a container
simcon stop <container-id>
```

## Project Structure

- `cmd/simcon`: Main CLI application
- `internal/container`: Core container implementation
- `internal/cgroups`: cgroups management
- `internal/filesystem`: Filesystem operations
- `internal/network`: Network namespace handling
- `pkg/utils`: Utility functions

## License

MIT 