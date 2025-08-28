# Go Code Similarity Detection Tool - Detailed Specification

## Overview

This document provides detailed technical specifications for **similarity-go**, a high-performance Go code similarity detection CLI tool that uses multi-factor AST analysis to identify duplicate and similar code patterns.

## System Requirements

### Runtime Requirements

- Go 1.21+ (uses generics)
- Memory: Minimum 512MB, Recommended 2GB+ for large projects
- CPU: Multi-core recommended for optimal parallel processing
- Storage: Minimal, primarily for configuration and temporary files

### Supported Platforms

- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64, arm64)

## Command Line Interface Specification

### Command Syntax

```bash
similarity-go [flags] <targets...>
```

### Flags and Options

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--threshold` | `-t` | float64 | 0.8 | Similarity threshold (0.0-1.0) |
| `--format` | `-f` | string | json | Output format (json\|yaml) |
| `--workers` | `-w` | int | 0 | Number of parallel workers (0=CPU count) |
| `--cache` | | bool | true | Enable result caching |
| `--config` | | string | | Custom configuration file path |
| `--output` | `-o` | string | | Output file path (default: stdout) |
| `--verbose` | `-v` | bool | false | Enable verbose logging |
| `--min-lines` | | int | 5 | Minimum function lines to analyze |
| `--help` | `-h` | bool | false | Show help information |
| `--version` | | bool | false | Show version information |

### Target Arguments

- **Files**: Individual Go source files (`.go` extension)
- **Directories**: Directory paths for recursive scanning
- **Mixed**: Combination of files and directories

### Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | File system error |
| 4 | Invalid arguments |

## Configuration Specification

### Configuration File Format

Configuration files use YAML format with the following structure:

```yaml
cli:
  default_threshold: 0.8
  default_min_lines: 5
  default_format: "json"
  default_workers: 0
  default_cache: true

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
    different_signature: 0.3
  limits:
    max_signature_length_diff: 50
    max_line_difference_ratio: 3.0
    max_cache_size: 10000

processing:
  max_empty_vs_populated: 5

output:
  refactor_suggestion: "Consider extracting common logic into a shared function"

ignore:
  default_file: ".similarityignore"
  patterns:
    - "*_test.go"
    - "testdata/"
    - "vendor/"
    - ".git/"
```

### Configuration File Discovery

The tool searches for configuration files in the following order:

1. File specified by `--config` flag
2. `.similarity-config.yaml` (current directory)
3. `.similarity-config.yml` (current directory)
4. `similarity-config.yaml` (current directory)
5. `similarity-config.yml` (current directory)

### Configuration Validation

All configuration values are validated at startup:

- **Thresholds**: Must be in range [0.0, 1.0]
- **Workers**: Must be >= 0 (0 = auto-detect CPU count)
- **Min Lines**: Must be > 0
- **Format**: Must be "json" or "yaml"
- **Cache Size**: Must be > 0
- **Line Difference Ratio**: Must be > 0.0

## File Processing Specification

### File Discovery

#### Supported File Extensions

- `.go` - Go source files (primary target)

#### Directory Traversal

- Recursive scanning of all subdirectories
- Follows symbolic links (with cycle detection)
- Respects ignore patterns during traversal

#### File Filtering

Files are excluded if they match any of the following criteria:

1. **Default Ignore Patterns**:
   - `*_test.go` - Go test files
   - `testdata/` - Test data directories
   - `vendor/` - Vendor dependencies
   - `.git/` - Git repositories

2. **Hidden Files**: Files/directories starting with `.` (except `.go` files)

3. **Custom Ignore Patterns**: Patterns defined in ignore files or configuration

### Ignore Pattern Specification

#### Ignore File Format

The `.similarityignore` file uses gitignore-compatible syntax:

```gitignore
# Comments start with #
*.pb.go                    # Protocol Buffer generated files
*_test.go                  # Test files
vendor/                    # Vendor directory
.git/                      # Git directory
generated/                 # Generated code directory
!important.pb.go           # Exception (negation with !)

# Directory patterns
**/build/                  # Any build directory
docs/**/*.md               # Markdown files in docs

# Specific files
config/secret.go           # Specific file path
```

#### Pattern Matching Rules

1. **Wildcards**:
   - `*` - Matches any characters except `/`
   - `**` - Matches any characters including `/`
   - `?` - Matches single character except `/`

2. **Directory Patterns**:
   - Patterns ending with `/` match directories only
   - Patterns without trailing `/` match both files and directories

3. **Negation**:
   - Patterns starting with `!` negate previous matches
   - Useful for creating exceptions to broader patterns

4. **Comments**:
   - Lines starting with `#` are treated as comments
   - Empty lines are ignored

## AST Processing Specification

### Function Extraction

#### Supported Function Types

1. **Regular Functions**: `func name() {}`
2. **Methods**: `func (r Receiver) name() {}`
3. **Generic Functions**: `func name[T any]() {}`
4. **Functions with Multiple Return Values**: `func name() (int, error) {}`

#### Excluded Function Types

1. **Interface Methods**: Method declarations in interfaces (no body)
2. **Functions Below Minimum Lines**: Functions shorter than `min_lines` threshold

### Function Metadata

Each extracted function includes:

```go
type Function struct {
    Name      string        // Function name
    File      string        // Source file path
    StartLine int          // Starting line number
    EndLine   int          // Ending line number
    AST       *ast.FuncDecl // Abstract syntax tree
    LineCount int          // Total lines of code
}
```

### Thread Safety

All function operations are thread-safe using `sync.RWMutex`:

- **Hash Calculation**: Lazy computation with double-checked locking
- **Signature Generation**: Thread-safe signature extraction
- **Concurrent Access**: Multiple goroutines can safely read function data

## Similarity Detection Specification

### Multi-Factor Algorithm

The similarity detection algorithm combines four weighted metrics:

#### 1. AST Tree Edit Distance (Weight: 30%)

**Algorithm**: Dynamic programming-based tree edit distance

**Process**:
1. Normalize AST nodes by removing variable names and literals
2. Calculate minimum edit distance between AST trees
3. Normalize by maximum tree size
4. Convert distance to similarity score (1.0 - normalized_distance)

**Formula**:
```
tree_similarity = 1.0 - (edit_distance / max(nodes1, nodes2))
```

#### 2. Token Sequence Analysis (Weight: 30%)

**Algorithm**: Levenshtein distance on normalized token sequences

**Process**:
1. Extract and normalize token sequences from AST
2. Remove identifiers, literals, and comments
3. Calculate Levenshtein distance between token sequences
4. Normalize by maximum sequence length

**Formula**:
```
token_similarity = 1.0 - (levenshtein_distance / max(len1, len2))
```

#### 3. Structural Signatures (Weight: 25%)

**Algorithm**: Structural pattern comparison

**Process**:
1. Generate structural signatures from function body
2. Extract control flow patterns (if, for, switch, etc.)
3. Compare signature strings for similarity
4. Use string similarity metrics (Levenshtein-based)

**Signature Components**:
- Control flow structures
- Function call patterns
- Variable declaration patterns
- Return statement patterns

#### 4. Function Signatures (Weight: 15%)

**Algorithm**: Function signature string comparison

**Process**:
1. Extract function signatures (name, parameters, return types)
2. Normalize parameter and return type names
3. Calculate string similarity between signatures
4. Apply signature-specific similarity thresholds

**Signature Format**:
```
func_name(param1_type, param2_type, ...) (return1_type, return2_type, ...)
```

### Weighted Score Calculation

The final similarity score combines all metrics:

```go
final_score = (tree_weight * tree_similarity) +
              (token_weight * token_similarity) +
              (structural_weight * structural_similarity) +
              (signature_weight * signature_similarity)
```

### Similarity Thresholds

#### Primary Threshold
- **CLI Threshold**: Functions must exceed this score to be considered similar
- **Default**: 0.8 (configurable via `--threshold` or configuration)

#### Algorithm-Specific Thresholds
- **Similar Operations**: 0.5 (minimum structural similarity)
- **Statement Count Penalty**: 0.5 (penalty for different statement counts)
- **Minimum Similarity**: 0.1 (absolute minimum for consideration)

### Performance Optimizations

#### Early Termination
- Skip expensive calculations if quick heuristics indicate low similarity
- Line count difference ratio check (configurable, default: 3.0)
- Signature length difference check (configurable, default: 50 characters)

#### Caching
- Function hash caching for identical function detection
- Signature caching for repeated function signature extraction
- Result caching for similarity score calculations

## Output Specification

### JSON Output Format

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
          "hash": "a1b2c3d4e5f6"
        },
        {
          "function": "ProcessAdmin",
          "file": "./internal/admin.go",
          "start_line": 15,
          "end_line": 30,
          "hash": "b2c3d4e5f6g7"
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

### YAML Output Format

```yaml
similar_groups:
  - id: "group_1"
    similarity_score: 0.95
    refactor_suggestion: "Consider extracting common logic into a shared function"
    functions:
      - function: "ProcessUser"
        file: "./internal/user.go"
        start_line: 10
        end_line: 25
        hash: "a1b2c3d4e5f6"
      - function: "ProcessAdmin"
        file: "./internal/admin.go"
        start_line: 15
        end_line: 30
        hash: "b2c3d4e5f6g7"

summary:
  similar_groups: 1
  total_duplications: 2
  total_functions: 45
```

### Field Descriptions

#### Similar Groups
- **id**: Unique identifier for the similarity group
- **similarity_score**: Average similarity score for the group (0.0-1.0)
- **refactor_suggestion**: Human-readable refactoring recommendation
- **functions**: Array of similar functions in the group

#### Function Information
- **function**: Function name
- **file**: Relative path to source file
- **start_line**: Starting line number in source file
- **end_line**: Ending line number in source file
- **hash**: Unique hash identifier for the function

#### Summary Statistics
- **similar_groups**: Total number of similarity groups found
- **total_duplications**: Total number of similar functions across all groups
- **total_functions**: Total number of functions analyzed

## Error Handling Specification

### Error Categories

#### 1. Configuration Errors
- Invalid configuration file syntax
- Invalid threshold values
- Invalid format specification
- Missing required configuration values

#### 2. File System Errors
- Inaccessible files or directories
- Permission denied errors
- File not found errors
- Invalid file paths

#### 3. Parse Errors
- Invalid Go syntax in source files
- Incomplete function declarations
- AST parsing failures

#### 4. Processing Errors
- Memory allocation failures
- Worker pool initialization errors
- Concurrent processing errors

### Error Reporting

#### Error Messages
All error messages include:
- **Context**: What operation was being performed
- **Location**: File path and line number (when applicable)
- **Cause**: Root cause of the error
- **Suggestion**: How to fix the error (when possible)

#### Error Format
```
Error: [CATEGORY] Description of what went wrong
  File: /path/to/file.go:123
  Cause: Detailed explanation of the root cause
  Suggestion: How to fix this error
```

#### Graceful Degradation
- Non-critical errors are logged and processing continues
- Critical errors cause graceful shutdown with cleanup
- Partial results are returned when possible

## Performance Specification

### Scalability Targets

#### Processing Capacity
- **Small Projects** (< 100 files): < 1 second
- **Medium Projects** (100-1000 files): < 10 seconds
- **Large Projects** (1000+ files): < 60 seconds

#### Memory Usage
- **Base Memory**: < 50MB for tool initialization
- **Per-Function Memory**: < 1KB per function analyzed
- **Maximum Memory**: Configurable via cache size limits

#### CPU Utilization
- **Default Workers**: Equal to CPU core count
- **CPU Efficiency**: > 80% CPU utilization during processing
- **Load Balancing**: Even work distribution across workers

### Performance Optimizations

#### Parallel Processing
- File parsing parallelization
- Similarity detection parallelization
- Worker pool with optimal queue management

#### Memory Management
- Efficient AST representation
- Garbage collection optimization
- Memory pool usage for frequent allocations

#### Algorithm Efficiency
- Early termination for low-similarity pairs
- Hash-based function deduplication
- Optimized tree edit distance algorithm

## Security Considerations

### Input Validation
- All file paths are validated and sanitized
- Configuration values are range-checked
- Command-line arguments are validated

### Resource Limits
- Maximum memory usage limits
- Maximum processing time limits
- Maximum file size limits

### Error Information Disclosure
- Error messages do not expose sensitive information
- File paths are sanitized in error output
- Stack traces are not exposed to end users

### Safe File Operations
- No modification of source files
- Read-only access to all input files
- Secure temporary file handling

## Testing Specification

### Test Coverage Requirements
- **Unit Tests**: > 75% line coverage per package
- **Integration Tests**: End-to-end workflow coverage
- **Performance Tests**: Benchmarking and profiling
- **Race Condition Tests**: Concurrent processing validation

### Test Categories

#### Unit Tests
- Individual function testing
- Mock-based dependency isolation
- Edge case and error condition testing
- Table-driven test patterns

#### Integration Tests
- Complete similarity detection workflows
- Configuration file processing
- Output format validation
- Error handling scenarios

#### Performance Tests
- Large project benchmarking
- Memory usage profiling
- CPU utilization measurement
- Scalability testing

#### Race Condition Tests
- Concurrent processing validation
- Thread safety verification
- Deadlock detection
- Data race prevention testing

## Version Compatibility

### API Stability
- Configuration format is backward compatible
- Command-line interface maintains compatibility
- Output format additions are non-breaking

### Migration Support
- Configuration file migration utilities
- Deprecation warnings for removed features
- Clear upgrade documentation

### Semantic Versioning
- **Major**: Breaking changes to API or output format
- **Minor**: New features, backward compatible
- **Patch**: Bug fixes, no API changes

This specification serves as the authoritative reference for **similarity-go** implementation and usage.