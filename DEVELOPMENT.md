# SimCon Development Guide

## Project Structure

```
simcon/
├── cmd/
│   └── simcon/              # Main application entry point
│       └── commands/        # CLI commands implementation
│           ├── create.go    # Container creation command
│           ├── delete.go    # Container deletion command
│           ├── init.go      # Container initialization command
│           ├── kill.go      # Container kill command
│           ├── list.go      # Container listing command
│           ├── run.go       # Container run command
│           ├── spec.go      # Container spec command
│           └── start.go     # Container start command
├── pkg/
│   ├── container/          # Core container implementation
│   │   ├── container.go    # Container interface and types
│   │   ├── container_linux.go  # Linux-specific container implementation
│   │   └── init.go         # Container initialization process
│   ├── cgroups/           # Cgroups management
│   │   └── cgroups.go     # Cgroups implementation
│   └── state/             # Container state management
│       └── state.go       # State implementation
└── vendor/                # External dependencies
```

## Core Components

### 1. Container Package (`pkg/container/`)

The container package implements the core container functionality:

- `container.go`: Defines the container interface and common types
- `container_linux.go`: Implements Linux-specific container operations
- `init.go`: Handles container initialization process

Key types:
```go
type Container struct {
    ID          string
    Bundle      string
    Process     *Process
    State       *ContainerState
    Spec        *specs.Spec
    InitProcess *InitProcess
}

type InitProcess struct {
    Container *Container
    cmd       *exec.Cmd
}
```

### 2. Cgroups Package (`pkg/cgroups/`)

Manages Linux cgroups for container resource control:

- CPU limits
- Memory limits
- Process limits
- Block IO limits
- Network class ID
- Device access
- Hugepages
- RDMA
- Unified cgroup limits

### 3. State Package (`pkg/state/`)

Manages container state persistence and retrieval.

## Development Guidelines

### 1. Code Organization

- Keep related functionality in the same package
- Use interfaces for better testability and modularity
- Follow Go's standard project layout
- Keep packages focused and cohesive

### 2. Error Handling

- Use custom error types for better error handling
- Wrap errors with context using `fmt.Errorf`
- Handle errors at the appropriate level
- Log errors with sufficient context

Example:
```go
type ContainerError struct {
    ID      string
    Op      string
    Message string
    Err     error
}

func (e *ContainerError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
    }
    return fmt.Sprintf("%s: %s", e.Op, e.Message)
}
```

### 3. Testing

- Write unit tests for all packages
- Use table-driven tests where appropriate
- Mock external dependencies
- Test error cases and edge conditions

Example:
```go
func TestContainerCreate(t *testing.T) {
    tests := []struct {
        name    string
        id      string
        bundle  string
        wantErr bool
    }{
        {
            name:    "valid container",
            id:      "test",
            bundle:  "/path/to/bundle",
            wantErr: false,
        },
        // Add more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            c, err := NewContainer(tt.id, tt.bundle)
            if (err != nil) != tt.wantErr {
                t.Errorf("NewContainer() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && c == nil {
                t.Error("NewContainer() returned nil container")
            }
        })
    }
}
```

### 4. Logging

- Use structured logging
- Include relevant context in log messages
- Use appropriate log levels
- Consider log rotation and management

### 5. Security

- Follow principle of least privilege
- Validate all inputs
- Handle sensitive data appropriately
- Use secure defaults

### 6. Performance

- Profile code for bottlenecks
- Use appropriate data structures
- Consider resource usage
- Implement proper cleanup

## Adding New Features

1. Create a new branch for the feature
2. Add tests for the new functionality
3. Implement the feature
4. Update documentation
5. Create a pull request

## Building and Testing

```bash
# Build the project
go build -o simcon ./cmd/simcon

# Run tests
go test ./...

# Run specific tests
go test ./pkg/container -v

# Run benchmarks
go test -bench=. ./pkg/container
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests
5. Submit a pull request

## Code Review Process

1. All code must be reviewed before merging
2. Tests must pass
3. Code must follow style guidelines
4. Documentation must be updated
5. Security implications must be considered

## Release Process

1. Update version numbers
2. Update changelog
3. Create release tag
4. Build release binaries
5. Create GitHub release

## Troubleshooting

Common issues and their solutions:

1. Container creation fails
   - Check bundle path
   - Verify permissions
   - Check system requirements

2. Container start fails
   - Check namespace support
   - Verify cgroup setup
   - Check process limits

3. Container networking issues
   - Verify network namespace
   - Check network configuration
   - Verify DNS settings 