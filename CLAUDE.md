# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go code similarity detection CLI tool that uses AST analysis to find duplicate and similar code patterns. The project uses Go 1.24.5 and follows the architectural patterns outlined in the extensive documentation in the `docs/` directory.

## Common Commands

### Development

- `go run .` - Run the application (main implementation not yet complete)
- `go build` - Build the binary
- `go mod tidy` - Clean up dependencies
- `go mod download` - Download dependencies

### Testing

- `go test ./...` - Run all tests
- `go test ./internal/ast/...` - Run AST package tests only
- `go test ./pkg/types/...` - Run types package tests only
- `go test -v ./...` - Run tests with verbose output
- `go test -race ./...` - Run tests with race detection
- `go test -cover ./...` - Run tests with coverage report

### Code Quality

- `go fmt ./...` - Format code
- `go vet ./...` - Run static analysis
- `golangci-lint run` - Run comprehensive linting (if available)

## Architecture

The codebase follows a layered architecture:

```text
├── cmd/           - CLI entry point (planned)
├── internal/      - Internal packages
│   ├── ast/       - AST parsing and function extraction
│   ├── scanner/   - File scanning with ignore patterns (planned)
│   ├── similarity/- Similarity detection algorithms (planned)
│   ├── cache/     - Caching system (planned)
│   └── worker/    - Parallel processing (planned)
├── pkg/
│   └── types/     - Utility types (Optional, Result)
└── docs/          - Comprehensive project documentation
```

## Key Components

### AST Package (`internal/ast/`)

- **Parser**: Parses Go files and extracts function declarations
- **Function**: Represents a Go function with metadata and AST representation
- Core functionality implemented, similarity detection planned

### Types Package (`pkg/types/`)

- **Optional[T]**: Rust-like Option type for null safety
- **Result[T]**: Rust-like Result type for error handling
- Fully implemented with comprehensive test coverage

## Current Implementation Status

**Completed:**

- Basic project structure and documentation
- AST parsing and function extraction (`internal/ast/parser.go`)
- Function representation with metadata (`internal/ast/function.go`)
- Utility types for error handling (`pkg/types/`)
- Test framework setup

**In Progress/Planned:**

- CLI interface (cobra-based)
- Similarity detection algorithms
- File scanning with ignore patterns
- Caching system
- Parallel processing
- Output formatting (JSON/YAML)

## Development Patterns

### Error Handling

The project uses Rust-inspired error handling:

```go
result := parser.ParseFile(filename)
if result.IsOk() {
    // Handle success
    parseResult := result.Unwrap()
} else {
    // Handle error
    err := result.Error()
}
```

### Function Analysis

Functions are represented with rich metadata:

```go
func := &Function{
    Name:      "exampleFunc",
    File:      "/path/to/file.go", 
    StartLine: 10,
    EndLine:   20,
    AST:       funcDeclNode,
    LineCount: 11,
}
```

## Testing Strategy

- Unit tests for each component with comprehensive coverage
- Table-driven tests following Go conventions
- Test data in `testdata/` directories (planned)
- Integration tests for end-to-end workflows (planned)

## MCP Tool Usage Guidelines

### Context7 MCP

Use context7 MCP for retrieving up-to-date library documentation:

1. **Resolve library ID first**: Always call `resolve-library-id` before `get-library-docs` unless user provides explicit library ID
2. **Focus searches**: Use specific topic parameters to get relevant documentation sections
3. **Token management**: Adjust token limits based on complexity - use higher values for complex integrations

### Serena MCP

Use serena MCP for efficient codebase analysis and editing:

1. **Symbolic analysis**: Prefer symbolic tools (`get_symbols_overview`, `find_symbol`) over reading entire files
2. **Targeted reading**: Use `include_body=true` only when necessary; explore structure first with `depth` parameter
3. **Memory usage**: Leverage project memories for architectural understanding before deep analysis
4. **Search patterns**: Use `search_for_pattern` for finding candidates before symbolic analysis
5. **Editing strategy**: Use symbol-based editing for complete functions/classes, regex-based for line-level changes

**Key Efficiency Rules:**

- Never read entire files without exploring symbols first
- Use `relative_path` parameters to restrict search scope
- Check memories for existing architectural knowledge
- Think about collected information before proceeding with complex operations

## Documentation

Extensive documentation available in `docs/`:

- `PROJECT_SUMMARY.md` - High-level overview and roadmap
- `ARCHITECTURE.md` - Detailed system architecture
- `SPECIFICATION.md` - Implementation specifications
- `IMPLEMENTATION.md` - Implementation guidelines

## Future Development

When implementing missing components:

1. Follow the detailed specifications in the `docs/` directory
2. Maintain the existing error handling patterns with Result/Optional types
3. Add comprehensive tests for new functionality
4. Use the planned parallel processing architecture for performance
5. Implement CLI using cobra framework as specified in the documentation

The project is well-architected with clear separation of concerns and comprehensive planning documentation to guide implementation.
