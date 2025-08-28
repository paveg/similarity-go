# similarity-go

[![CI](https://github.com/paveg/similarity-go/workflows/CI/badge.svg)](https://github.com/paveg/similarity-go/actions)
[![codecov](https://codecov.io/gh/paveg/similarity-go/graph/badge.svg?token=IM08X5VLQX)](https://codecov.io/gh/paveg/similarity-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/paveg/similarity-go)](https://goreportcard.com/report/github.com/paveg/similarity-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/paveg/similarity-go.svg)](https://pkg.go.dev/github.com/paveg/similarity-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A high-performance Go code similarity detection CLI tool that uses multi-factor AST analysis to find duplicate and similar code patterns with advanced similarity detection algorithms.

## Features

- **Multi-Factor Similarity Detection**: Combines AST tree edit distance, token sequence analysis, structural signatures, and signature matching with weighted scoring
- **Advanced AST Analysis**: Deep structural analysis using Go's abstract syntax tree with normalized comparison
- **Intelligent Directory Scanning**: Recursive traversal with smart filtering for Go files, excluding vendor/, hidden files, and build directories
- **High-Performance Parallel Processing**: CPU-efficient worker pools with concurrent similarity detection
- **Thread-Safe Operations**: Race condition-free concurrent processing with proper synchronization
- **Configurable Similarity Thresholds**: Fine-tuned similarity detection with adjustable thresholds (0.0-1.0)
- **Multiple Output Formats**: JSON and YAML output with structured similarity reports
- **Comprehensive Test Coverage**: 78-88% test coverage with extensive unit and integration tests
- **Configuration Management**: YAML-based configuration with validation and fallback defaults
- **Generic Math Utilities**: Type-safe mathematical functions using Go 1.21+ generics

## Installation

### Using Go Install (Recommended)

```bash
go install github.com/paveg/similarity-go/cmd/similarity-go@latest
```

This will install the binary to your `$GOPATH/bin` or `$GOBIN` directory.

### Using Homebrew (macOS/Linux)

```bash
brew tap paveg/tap
brew install similarity-go
```

### Download Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/paveg/similarity-go/releases/latest).

### From Source

```bash
git clone https://github.com/paveg/similarity-go.git
cd similarity-go
make build        # Build for current platform
```

The binary will be available as `bin/similarity-go`.

### Docker

```bash
docker run --rm -v $(pwd):/workspace paveg/similarity-go --threshold 0.8 /workspace
```

## Usage

```bash
# Basic usage - scan entire directory
./similarity-go ./src

# Analyze specific files
./similarity-go main.go utils.go

# Multiple directories and mixed targets
./similarity-go ./cmd ./internal main.go

# With custom threshold and output format
./similarity-go --threshold 0.7 --format yaml ./project

# Save results to file with verbose output
./similarity-go --verbose --output results.json ./codebase
```

### Command Line Options

- `--threshold, -t`: Similarity threshold (0.0-1.0, default: 0.8)
- `--format, -f`: Output format (json|yaml, default: json)
- `--workers, -w`: Number of parallel workers (default: CPU count)  
- `--cache`: Enable result caching (default: true)
- `--config`: Custom configuration file path
- `--output, -o`: Output file (default: stdout)
- `--verbose, -v`: Enable verbose logging
- `--min-lines`: Minimum function lines to analyze (default: 5)

## Output Format

The tool generates structured JSON/YAML reports with detailed similarity analysis:

```json
{
  "similar_groups": [
    {
      "id": "group_1",
      "similarity_score": 0.95,
      "refactor_suggestion": "Consider extracting common logic into a shared function",
      "functions": [
        {
          "function": "ProcessUser",
          "file": "./internal/user.go",
          "start_line": 10,
          "end_line": 25,
          "hash": "a1b2c3d4"
        },
        {
          "function": "ProcessAdmin", 
          "file": "./internal/admin.go",
          "start_line": 15,
          "end_line": 30,
          "hash": "e5f6g7h8"
        }
      ]
    }
  ],
  "summary": {
    "similar_groups": 1,
    "total_duplications": 2,
    "total_functions": 45
  }
}
```

## Development

### Prerequisites

- Go 1.21+ (uses generics)
- golangci-lint for linting

### Development Commands

```bash
go test ./...              # Run all tests
go test -cover ./...       # Run tests with coverage
go test -race ./...        # Run tests with race detection
make lint                  # Run linter
make test                  # Run tests and linting
```

### Testing

The project maintains high test coverage with comprehensive test suites:

- **Similarity package**: 78.0% coverage
- **Command package**: 87.8% coverage  
- **Config package**: 81.4% coverage
- **Full test suite**: Race condition detection and parallel processing tests

## Architecture

```text
├── cmd/                   # CLI entry point and command handling
├── internal/              # Internal packages
│   ├── ast/              # AST parsing and function extraction
│   ├── similarity/       # Multi-factor similarity detection algorithms  
│   ├── config/           # Configuration management and validation
│   ├── worker/           # Parallel processing and worker pools
│   └── test-helpers/     # Test utilities and helpers
├── pkg/                  # Public reusable packages
│   ├── math-util/        # Generic math utilities (Min, Max, Abs)
│   └── types/            # Utility types (Optional, Result)
└── testdata/             # Test fixtures and sample data
```

### Similarity Detection Algorithm

The tool uses a sophisticated multi-factor approach:

1. **AST Tree Edit Distance**: Structural comparison using dynamic programming
2. **Token Sequence Analysis**: Normalized token similarity using Levenshtein distance  
3. **Structural Signatures**: Function signature and body structure comparison
4. **Weighted Scoring**: Combines multiple similarity metrics with configurable weights

Default algorithm weights:

- Tree Edit Distance: 30%
- Token Similarity: 30%
- Structural Analysis: 25%
- Signature Matching: 15%

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure tests pass (`go test ./...`)
5. Run linting (`make lint`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## Configuration

Create a `.similarity-config.yaml` file in your project root:

```yaml
cli:
  default_threshold: 0.8
  default_min_lines: 5
  default_format: "json"
  default_workers: 0  # 0 = use all CPU cores

similarity:
  thresholds:
    default_similar_operations: 0.5
    statement_count_penalty: 0.5
    min_similarity: 0.1
  weights:
    tree_edit: 0.3
    token_similarity: 0.3
    structural: 0.25
    signature: 0.15
  limits:
    max_cache_size: 10000
    max_line_difference_ratio: 3.0

ignore:
  default_file: ".similarityignore"
  patterns:
    - "*_test.go"
    - "testdata/"
    - "vendor/"
    - ".git/"
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Note**: This tool is actively maintained and continuously improved based on real-world usage and feedback.
