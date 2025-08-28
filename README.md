# similarity-go

[![CI](https://github.com/paveg/similarity-go/workflows/CI/badge.svg)](https://github.com/paveg/similarity-go/actions)
[![codecov](https://codecov.io/gh/paveg/similarity-go/graph/badge.svg?token=IM08X5VLQX)](https://codecov.io/gh/paveg/similarity-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/paveg/similarity-go)](https://goreportcard.com/report/github.com/paveg/similarity-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Go code similarity detection CLI tool that uses AST analysis to find duplicate and similar code patterns.

## Features

- **AST-based Analysis**: Uses Go's abstract syntax tree for accurate code similarity detection
- **Directory Scanning**: Recursive traversal of directories with intelligent file filtering
- **Configurable Thresholds**: Adjust similarity thresholds to match your needs
- **Multiple Output Formats**: Support for JSON and YAML output formats
- **Ignore Patterns**: Built-in filtering for vendor/, hidden files, and build directories
- **Mixed Target Support**: Analyze individual files and directories in single command
- **Comprehensive Testing**: 67-100% test coverage with comprehensive test suite

## Installation

### From Source

```bash
git clone https://github.com/paveg/similarity-go.git
cd similarity-go
make build
```

The binary will be available in `bin/similarity-go`.

### Cross-Platform Builds

```bash
make build-all    # Build for all platforms
make build-linux  # Linux (amd64, arm64)
make build-darwin # macOS (amd64, arm64)
make build-windows # Windows (amd64, arm64)
```

## Usage

```bash
# Basic usage - scan entire directory
./bin/similarity-go ./src

# Analyze specific files
./bin/similarity-go main.go utils.go

# Multiple directories
./bin/similarity-go ./cmd ./internal

# With custom threshold and output format
./bin/similarity-go --threshold 0.8 --format yaml ./project

# Verbose output with detailed file listing
./bin/similarity-go --verbose --output results.json ./code
```

### Options

- `--threshold, -t`: Similarity threshold (0.0-1.0, default: 0.8)
- `--format, -f`: Output format (json|yaml, default: json)
- `--workers, -w`: Number of parallel workers (default: CPU count)
- `--cache`: Enable caching (default: true)
- `--ignore`: Ignore file path (default: .similarityignore)
- `--output, -o`: Output file (default: stdout)
- `--verbose, -v`: Verbose output
- `--min-lines`: Minimum function lines to analyze (default: 5)

## Development

### Prerequisites

- Go 1.24.5 or higher
- golangci-lint v2.4.0 or higher

### Development Commands

```bash
make help          # Show all available commands
make dev           # Format and test (quick development cycle)
make quality       # Full quality check (format, vet, lint, test, coverage)
make test-coverage # Run tests with coverage
make coverage-html # Generate HTML coverage report
```

### Testing

The project follows TDD principles with 80%+ code coverage requirement:

```bash
make test                    # Run all tests
make test-coverage           # Run with coverage
make coverage-check          # Verify coverage threshold
```

## Architecture

```text
├── cmd/           # CLI entry point with directory scanning
├── internal/      # Internal packages
│   ├── ast/       # AST parsing and function extraction
│   ├── similarity/# Multi-factor similarity detection algorithms
│   ├── testhelpers/# Test utilities and helpers
│   ├── cache/     # Caching system (planned)
│   └── worker/    # Parallel processing (planned)
├── pkg/           # Public packages
│   ├── mathutil/  # Generic math utilities (Min, Max, Abs)
│   └── types/     # Utility types (Optional, Result)
└── testdata/      # Test data and fixtures
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for your changes
4. Ensure 80%+ test coverage
5. Run `make quality` to verify code quality
6. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
