# Go Code Similarity Detection Tool - Detailed Specification

## Ignore Files処理設計

### .similarityignore ファイル形式

```
# コメント行（#で始まる）
*.pb.go                    # Protocol Buffer生成ファイル
*_test.go                  # テストファイル
vendor/                    # vendorディレクトリ
.git/                      # Gitディレクトリ
node_modules/              # Node.jsモジュール
*.min.js                   # 最小化されたJavaScript
generated/                 # 生成されたコードディレクトリ
!important.pb.go           # 除外の例外（!で始まる）

# ディレクトリパターン
**/build/                  # 任意の階層のbuildディレクトリ
docs/**/*.md               # docsディレクトリ以下のMarkdownファイル

# 特定ファイル
config/secret.go           # 特定のファイル
```

### Ignore処理実装

```go
// internal/scanner/ignore.go
package scanner

import (
    "bufio"
    "os"
    "path/filepath"
    "regexp"
    "strings"
)

type IgnoreMatcher struct {
    patterns    []ignorePattern
    basePath    string
}

type ignorePattern struct {
    pattern   string
    regex     *regexp.Regexp
    negate    bool
    dirOnly   bool
}

func NewIgnoreMatcher(ignoreFile, basePath string) (*IgnoreMatcher, error) {
    matcher := &IgnoreMatcher{
        basePath: basePath,
    }
    
    if err := matcher.loadIgnoreFile(ignoreFile); err != nil {
        return nil, err
    }
    
    // Add default patterns
    matcher.addDefaultPatterns()
    
    return matcher, nil
}

func (im *IgnoreMatcher) ShouldIgnore(path string) bool {
    relPath, err := filepath.Rel(im.basePath, path)
    if err != nil {
        return false
    }
    
    ignored := false
    
    for _, pattern := range im.patterns {
        if pattern.matches(relPath) {
            if pattern.negate {
                ignored = false
            } else {
                ignored = true
            }
        }
    }
    
    return ignored
}

func (im *IgnoreMatcher) loadIgnoreFile(ignoreFile string) error {
    file, err := os.Open(ignoreFile)
    if err != nil {
        if os.IsNotExist(err) {
            return nil // Ignore file doesn't exist, that's OK
        }
        return err
    }
    defer file.Close()
    
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        
        // Skip empty lines and comments
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        
        pattern := im.parsePattern(line)
        im.patterns = append(im.patterns, pattern)
    }
    
    return scanner.Err()
}

func (im *IgnoreMatcher) parsePattern(line string) ignorePattern {
    pattern := ignorePattern{
        pattern: line,
    }
    
    // Handle negation
    if strings.HasPrefix(line, "!") {
        pattern.negate = true
        line = line[1:]
    }
    
    // Handle directory-only patterns
    if strings.HasSuffix(line, "/") {
        pattern.dirOnly = true
        line = line[:len(line)-1]
    }
    
    // Convert glob pattern to regex
    pattern.regex = im.globToRegex(line)
    
    return pattern
}

func (im *IgnoreMatcher) globToRegex(glob string) *regexp.Regexp {
    // Convert glob patterns to regex
    regex := strings.ReplaceAll(glob, ".", "\\.")
    regex = strings.ReplaceAll(regex, "*", "[^/]*")
    regex = strings.ReplaceAll(regex, "**", ".*")
    regex = "^" + regex + "$"
    
    return regexp.MustCompile(regex)
}

func (p *ignorePattern) matches(path string) bool {
    return p.regex.MatchString(path)
}
```

## 出力形式設計

### JSON出力形式

```go
// internal/output/types.go
package output

import (
    "time"
)

type SimilarityReport struct {
    Metadata     ReportMetadata  `json:"metadata"`
    Summary      Summary         `json:"summary"`
    SimilarGroups []SimilarGroup `json:"similar_groups"`
    Statistics   Statistics      `json:"statistics,omitempty"`
}

type ReportMetadata struct {
    Version     string    `json:"version"`
    GeneratedAt time.Time `json:"generated_at"`
    Tool        string    `json:"tool"`
    Config      Config    `json:"config"`
}

type Config struct {
    Threshold     float64  `json:"threshold"`
    MinLines      int      `json:"min_lines"`
    Workers       int      `json:"workers"`
    CacheEnabled  bool     `json:"cache_enabled"`
    Targets       []string `json:"targets"`
    IgnoreFile    string   `json:"ignore_file"`
}

type Summary struct {
    TotalFiles        int     `json:"total_files"`
    ProcessedFiles    int     `json:"processed_files"`
    TotalFunctions    int     `json:"total_functions"`
    SimilarGroups     int     `json:"similar_groups"`
    TotalDuplications int     `json:"total_duplications"`
    ProcessingTime    string  `json:"processing_time"`
    AverageSimilarity float64 `json:"average_similarity"`
}

type SimilarGroup struct {
    ID                 string        `json:"id"`
    SimilarityScore    float64       `json:"similarity_score"`
    Type               string        `json:"type"`
    Functions          []FunctionRef `json:"functions"`
    RefactorSuggestion string        `json:"refactor_suggestion"`
    Impact             Impact        `json:"impact"`
}

type FunctionRef struct {
    File       string            `json:"file"`
    Function   string            `json:"function"`
    StartLine  int               `json:"start_line"`
    EndLine    int               `json:"end_line"`
    LineCount  int               `json:"line_count"`
    Hash       string            `json:"hash"`
    Signature  string            `json:"signature,omitempty"`
    Complexity int               `json:"complexity,omitempty"`
    Metadata   map[string]string `json:"metadata,omitempty"`
}

type Impact struct {
    EstimatedLines   int     `json:"estimated_lines"`
    ComplexityScore  float64 `json:"complexity_score"`
    MaintenanceRisk  string  `json:"maintenance_risk"`
    RefactorPriority string  `json:"refactor_priority"`
}

type Statistics struct {
    SimilarityDistribution map[string]int    `json:"similarity_distribution"`
    FileTypeDistribution   map[string]int    `json:"file_type_distribution"`
    FunctionSizeStats      SizeStatistics    `json:"function_size_stats"`
    ProcessingStats        ProcessingStats   `json:"processing_stats"`
}

type SizeStatistics struct {
    Min     int     `json:"min"`
    Max     int     `json:"max"`
    Average float64 `json:"average"`
    Median  int     `json:"median"`
}

type ProcessingStats struct {
    ParsingTime     string `json:"parsing_time"`
    ComparisonTime  string `json:"comparison_time"`
    CacheHitRate    float64 `json:"cache_hit_rate"`
    FilesPerSecond  float64 `json:"files_per_second"`
}
```

### YAML出力実装

```go
// internal/output/formatter.go
package output

import (
    "encoding/json"
    "gopkg.in/yaml.v3"
    "io"
)

type Formatter interface {
    Format(report *SimilarityReport, writer io.Writer) error
}

type JSONFormatter struct {
    indent bool
}

func NewJSONFormatter(indent bool) *JSONFormatter {
    return &JSONFormatter{indent: indent}
}

func (jf *JSONFormatter) Format(report *SimilarityReport, writer io.Writer) error {
    encoder := json.NewEncoder(writer)
    if jf.indent {
        encoder.SetIndent("", "  ")
    }
    return encoder.Encode(report)
}

type YAMLFormatter struct{}

func NewYAMLFormatter() *YAMLFormatter {
    return &YAMLFormatter{}
}

func (yf *YAMLFormatter) Format(report *SimilarityReport, writer io.Writer) error {
    encoder := yaml.NewEncoder(writer)
    defer encoder.Close()
    return encoder.Encode(report)
}

type FormatterFactory struct{}

func (ff *FormatterFactory) Create(format string, indent bool) Formatter {
    switch format {
    case "yaml", "yml":
        return NewYAMLFormatter()
    case "json":
        fallthrough
    default:
        return NewJSONFormatter(indent)
    }
}
```

## エラーハンドリング・ログ設計

### エラー分類・処理

```go
// internal/errors/types.go
package errors

import (
    "fmt"
)

type ErrorType int

const (
    ParseError ErrorType = iota
    FileSystemError
    CacheError
    ConfigurationError
    ThresholdError
    ComparisonError
    OutputError
)

type SimilarityError struct {
    Type      ErrorType
    Message   string
    File      string
    Line      int
    Function  string
    Cause     error
    Context   map[string]interface{}
}

func (se *SimilarityError) Error() string {
    if se.File != "" {
        return fmt.Sprintf("%s:%d: %s", se.File, se.Line, se.Message)
    }
    return se.Message
}

func (se *SimilarityError) Unwrap() error {
    return se.Cause
}

// エラーファクトリー関数
func NewParseError(file string, line int, cause error) *SimilarityError {
    return &SimilarityError{
        Type:    ParseError,
        Message: "failed to parse Go file",
        File:    file,
        Line:    line,
        Cause:   cause,
    }
}

func NewFileSystemError(file string, cause error) *SimilarityError {
    return &SimilarityError{
        Type:    FileSystemError,
        Message: "file system operation failed",
        File:    file,
        Cause:   cause,
    }
}
```

### ログ設計

```go
// internal/logger/logger.go
package logger

import (
    "io"
    "log/slog"
    "os"
)

type Logger struct {
    *slog.Logger
    level   slog.Level
    verbose bool
}

func New(verbose bool, output io.Writer) *Logger {
    level := slog.LevelInfo
    if verbose {
        level = slog.LevelDebug
    }
    
    if output == nil {
        output = os.Stderr
    }
    
    handler := slog.NewTextHandler(output, &slog.HandlerOptions{
        Level: level,
    })
    
    return &Logger{
        Logger:  slog.New(handler),
        level:   level,
        verbose: verbose,
    }
}

func (l *Logger) Progress(message string, current, total int) {
    if l.verbose {
        l.Info("progress", 
            slog.String("message", message),
            slog.Int("current", current),
            slog.Int("total", total),
            slog.Float64("percentage", float64(current)/float64(total)*100),
        )
    }
}

func (l *Logger) Performance(operation string, duration string, details map[string]interface{}) {
    if l.verbose {
        attrs := []slog.Attr{
            slog.String("operation", operation),
            slog.String("duration", duration),
        }
        
        for k, v := range details {
            attrs = append(attrs, slog.Any(k, v))
        }
        
        l.LogAttrs(nil, slog.LevelDebug, "performance", attrs...)
    }
}
```

## テスト戦略

### テストディレクトリ構造

```
testdata/
├── samples/                    # サンプルコード
│   ├── identical/             # 完全に同一な関数
│   ├── similar/               # 類似度の高い関数
│   ├── different/             # 類似度の低い関数
│   └── edge_cases/            # エッジケース
├── configs/                   # テスト用設定ファイル
├── expected/                  # 期待される出力結果
└── benchmarks/                # ベンチマーク用大規模コード
```

### テストカテゴリ

```go
// tests/unit/ast_test.go
func TestASTParser(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected int // 期待される関数数
    }{
        {
            name: "single function",
            input: `package main
func hello() {
    fmt.Println("hello")
}`,
            expected: 1,
        },
        // その他のテストケース...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // テスト実装
        })
    }
}

// tests/integration/similarity_test.go
func TestSimilarityDetection(t *testing.T) {
    testCases := []struct {
        name      string
        threshold float64
        input     []string
        expected  int // 期待されるグループ数
    }{
        {
            name:      "identical functions",
            threshold: 0.9,
            input:     []string{"testdata/samples/identical/func1.go", "testdata/samples/identical/func2.go"},
            expected:  1,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // 統合テスト実装
        })
    }
}

// tests/benchmark/performance_test.go
func BenchmarkSimilarityDetection(b *testing.B) {
    // 大規模プロジェクトでのパフォーマンステスト
    for i := 0; i < b.N; i++ {
        // ベンチマーク実装
    }
}
```

## パフォーマンス最適化

### メモリ最適化戦略

1. **AST構造の最適化**
   - 不要なメタデータの削除
   - 構造体のメモリアライメント最適化
   - プールを使用したオブジェクト再利用

2. **ガベージコレクション最適化**
   - 大きなオブジェクトの早期解放
   - 循環参照の回避
   - メモリプロファイリング

### CPU最適化戦略

1. **並列処理の最適化**
   - CPU効率的なワーカープール
   - ロックフリーデータ構造
   - goroutineリーク防止

2. **アルゴリズム最適化**
   - 早期終了条件の実装
   - キャッシュフレンドリーなデータ構造
   - SIMD命令の活用（可能な場合）

### パフォーマンス測定

```go
// internal/profiler/profiler.go
package profiler

import (
    "runtime"
    "time"
)

type Profiler struct {
    startTime time.Time
    startMem  runtime.MemStats
}

func New() *Profiler {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    return &Profiler{
        startTime: time.Now(),
        startMem:  m,
    }
}

func (p *Profiler) Report() ProfileReport {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    return ProfileReport{
        Duration:       time.Since(p.startTime),
        MemoryUsed:     m.Alloc - p.startMem.Alloc,
        PeakMemory:     m.Sys,
        GCCycles:       m.NumGC - p.startMem.NumGC,
        Goroutines:     runtime.NumGoroutine(),
    }
}

type ProfileReport struct {
    Duration   time.Duration
    MemoryUsed uint64
    PeakMemory uint64
    GCCycles   uint32
    Goroutines int
}
```

## ドキュメント作成計画

### ドキュメント構成

```
docs/
├── README.md                  # プロジェクト概要
├── INSTALLATION.md            # インストール手順
├── USAGE.md                   # 使用方法・例
├── CONFIGURATION.md           # 設定詳細
├── API.md                     # API仕様（将来の拡張用）
├── ALGORITHMS.md              # アルゴリズム詳細
├── CONTRIBUTING.md            # 貢献ガイドライン
├── CHANGELOG.md               # 変更履歴
└── examples/                  # 使用例
    ├── basic/                 # 基本的な使用例
    ├── advanced/              # 高度な使用例
    └── ci-integration/        # CI/CD統合例
```

### 使用例

```bash
# 基本的な使用
similarity-go ./src

# 閾値を指定
similarity-go --threshold 0.8 ./src

# YAML出力
similarity-go --format yaml --output report.yaml ./src

# 詳細出力
similarity-go --verbose --workers 8 ./src

# 特定のファイルのみ
similarity-go main.go utils.go handler.go

# カスタムignoreファイル
similarity-go --ignore .myignore ./project
```

これで、Go ASTを利用したコード類似性検証CLIツールの包括的な設計が完成しました。
