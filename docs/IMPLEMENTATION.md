# Implementation Guide - Go Code Similarity Detection Tool

## 実装フェーズ詳細

### Phase 1: プロジェクト基盤実装

#### 1.1 プロジェクト初期化

```bash
# プロジェクト作成
mkdir similarity-go
cd similarity-go
go mod init github.com/paveg/similarity-go

# 依存関係追加
go get github.com/spf13/cobra@latest
go get gopkg.in/yaml.v3@latest
go get github.com/spf13/viper@latest
```

#### 1.2 ディレクトリ構造作成

```
similarity-go/
├── cmd/
│   └── main.go                 # エントリーポイント
├── internal/
│   ├── ast/
│   │   ├── parser.go          # AST解析
│   │   ├── function.go        # 関数抽出
│   │   ├── normalizer.go      # AST正規化
│   │   └── hasher.go          # 構造ハッシュ
│   ├── similarity/
│   │   ├── detector.go        # 類似性検出
│   │   ├── algorithm.go       # 比較アルゴリズム
│   │   └── threshold.go       # 閾値処理
│   ├── scanner/
│   │   ├── walker.go          # ファイル走査
│   │   └── ignore.go          # ignore処理
│   ├── cache/
│   │   ├── manager.go         # キャッシュ管理
│   │   └── storage.go         # ストレージ実装
│   ├── worker/
│   │   └── pool.go            # ワーカープール
│   ├── output/
│   │   ├── formatter.go       # 出力フォーマット
│   │   └── types.go           # 出力型定義
│   └── config/
│       └── config.go          # 設定管理
├── pkg/
│   └── types/
│       └── similarity.go      # 公開型定義
├── testdata/
│   ├── sample1/               # テスト用サンプルコード
│   └── sample2/
├── .gitignore
├── .similarity.yaml           # デフォルト設定
├── Makefile
└── README.md
```

#### 1.3 基本CLIフレームワーク実装

```go
// cmd/main.go
package main

import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    "github.com/paveg/similarity-go/internal/config"
)

var (
    cfgFile    string
    threshold  float64
    format     string
    workers    int
    useCache   bool
    ignoreFile string
    output     string
    verbose    bool
    minLines   int
)

var rootCmd = &cobra.Command{
    Use:   "similarity-go [flags] <targets...>",
    Short: "Go code similarity detection tool",
    Long:  `Detect similar code patterns in Go projects using AST analysis`,
    Args:  cobra.MinimumNArgs(1),
    RunE:  runSimilarityCheck,
}

func init() {
    cobra.OnInitialize(initConfig)
    
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
    rootCmd.Flags().Float64VarP(&threshold, "threshold", "t", 0.7, "similarity threshold (0.0-1.0)")
    rootCmd.Flags().StringVarP(&format, "format", "f", "json", "output format (json|yaml)")
    rootCmd.Flags().IntVarP(&workers, "workers", "w", 0, "number of workers (0=auto)")
    rootCmd.Flags().BoolVar(&useCache, "cache", true, "enable caching")
    rootCmd.Flags().StringVar(&ignoreFile, "ignore", ".similarityignore", "ignore file")
    rootCmd.Flags().StringVarP(&output, "output", "o", "", "output file")
    rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
    rootCmd.Flags().IntVar(&minLines, "min-lines", 5, "minimum function lines")
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### Phase 2: AST解析実装

#### 2.1 基本パーサー実装

```go
// internal/ast/parser.go
package ast

import (
    "go/ast"
    "go/parser"
    "go/token"
    "path/filepath"
)

type Parser struct {
    fileSet *token.FileSet
}

func NewParser() *Parser {
    return &Parser{
        fileSet: token.NewFileSet(),
    }
}

type ParseResult struct {
    File      *ast.File
    Functions []*Function
    Error     error
}

func (p *Parser) ParseFile(filename string) (*ParseResult, error) {
    src, err := os.ReadFile(filename)
    if err != nil {
        return &ParseResult{Error: err}, err
    }
    
    file, err := parser.ParseFile(p.fileSet, filename, src, parser.ParseComments)
    if err != nil {
        return &ParseResult{Error: err}, err
    }
    
    functions := p.extractFunctions(file, filename)
    
    return &ParseResult{
        File:      file,
        Functions: functions,
    }, nil
}

func (p *Parser) extractFunctions(file *ast.File, filename string) []*Function {
    var functions []*Function
    
    ast.Inspect(file, func(n ast.Node) bool {
        switch node := n.(type) {
        case *ast.FuncDecl:
            if node.Body != nil { // Skip function declarations without body
                fn := &Function{
                    Name:      node.Name.Name,
                    File:      filename,
                    AST:       node,
                    StartLine: p.fileSet.Position(node.Pos()).Line,
                    EndLine:   p.fileSet.Position(node.End()).Line,
                }
                functions = append(functions, fn)
            }
        }
        return true
    })
    
    return functions
}
```

#### 2.2 関数構造体定義

```go
// internal/ast/function.go
package ast

import (
    "go/ast"
    "go/format"
    "bytes"
)

type Function struct {
    Name       string
    File       string
    StartLine  int
    EndLine    int
    AST        *ast.FuncDecl
    Normalized *ast.FuncDecl
    Hash       string
    Signature  string
    LineCount  int
}

func (f *Function) GetSignature() string {
    if f.Signature != "" {
        return f.Signature
    }
    
    var buf bytes.Buffer
    if f.AST.Type != nil {
        format.Node(&buf, token.NewFileSet(), f.AST.Type)
        f.Signature = buf.String()
    }
    
    return f.Signature
}

func (f *Function) GetSource() (string, error) {
    var buf bytes.Buffer
    err := format.Node(&buf, token.NewFileSet(), f.AST)
    if err != nil {
        return "", err
    }
    return buf.String(), nil
}

func (f *Function) IsValid(minLines int) bool {
    return f.LineCount >= minLines && f.AST.Body != nil
}
```

#### 2.3 AST正規化実装

```go
// internal/ast/normalizer.go
package ast

import (
    "go/ast"
    "go/token"
    "strconv"
)

type Normalizer struct {
    ignoreNames     bool
    ignoreComments  bool
    ignoreLiterals  bool
}

func NewNormalizer() *Normalizer {
    return &Normalizer{
        ignoreNames:     true,
        ignoreComments:  true,
        ignoreLiterals:  false,
    }
}

func (n *Normalizer) Normalize(fn *ast.FuncDecl) *ast.FuncDecl {
    // Deep copy the AST node
    normalized := n.deepCopyFuncDecl(fn)
    
    // Apply normalization transformations
    ast.Inspect(normalized, func(node ast.Node) bool {
        switch typed := node.(type) {
        case *ast.Ident:
            if n.ignoreNames && typed.Obj != nil {
                // Replace variable names with generic names
                typed.Name = n.generateGenericName(typed.Name)
            }
        case *ast.BasicLit:
            if n.ignoreLiterals {
                // Replace literals with generic values
                typed.Value = n.generateGenericLiteral(typed.Kind)
            }
        case *ast.CommentGroup:
            if n.ignoreComments {
                // Remove comments
                return false
            }
        }
        return true
    })
    
    return normalized
}

func (n *Normalizer) generateGenericName(original string) string {
    // Generate consistent generic names based on usage patterns
    // e.g., variables -> "var1", "var2", functions -> "func1", "func2"
    return "var" // Simplified for example
}

func (n *Normalizer) generateGenericLiteral(kind token.Token) string {
    switch kind {
    case token.STRING:
        return `""`
    case token.INT:
        return "0"
    case token.FLOAT:
        return "0.0"
    default:
        return "nil"
    }
}
```

#### 2.4 構造ハッシュ実装

```go
// internal/ast/hasher.go
package ast

import (
    "crypto/sha256"
    "fmt"
    "go/ast"
    "go/token"
    "sort"
    "strings"
)

type StructureHasher struct {
    includeTypes     bool
    includeOperators bool
    includeFlow      bool
}

func NewStructureHasher() *StructureHasher {
    return &StructureHasher{
        includeTypes:     true,
        includeOperators: true,
        includeFlow:      true,
    }
}

func (sh *StructureHasher) HashFunction(fn *ast.FuncDecl) (string, error) {
    features := sh.extractFeatures(fn)
    
    // Sort features for consistent hashing
    sort.Strings(features)
    
    combined := strings.Join(features, "|")
    hash := sha256.Sum256([]byte(combined))
    
    return fmt.Sprintf("%x", hash), nil
}

func (sh *StructureHasher) extractFeatures(node ast.Node) []string {
    var features []string
    
    ast.Inspect(node, func(n ast.Node) bool {
        if n == nil {
            return false
        }
        
        switch typed := n.(type) {
        case *ast.IfStmt:
            features = append(features, "if")
        case *ast.ForStmt:
            features = append(features, "for")
        case *ast.RangeStmt:
            features = append(features, "range")
        case *ast.SwitchStmt:
            features = append(features, "switch")
        case *ast.CallExpr:
            features = append(features, "call")
        case *ast.BinaryExpr:
            if sh.includeOperators {
                features = append(features, "binary:"+typed.Op.String())
            }
        case *ast.UnaryExpr:
            if sh.includeOperators {
                features = append(features, "unary:"+typed.Op.String())
            }
        case *ast.ReturnStmt:
            features = append(features, "return")
        case *ast.AssignStmt:
            features = append(features, "assign:"+typed.Tok.String())
        }
        
        return true
    })
    
    return features
}
```

### Phase 3: 類似性検出実装

#### 3.1 検出エンジン実装

```go
// internal/similarity/detector.go
package similarity

import (
    "github.com/paveg/similarity-go/internal/ast"
)

type Detector struct {
    threshold float64
    algorithm ComparisonAlgorithm
    hasher    *ast.StructureHasher
}

func NewDetector(threshold float64) *Detector {
    return &Detector{
        threshold: threshold,
        algorithm: NewStructuralComparison(),
        hasher:    ast.NewStructureHasher(),
    }
}

type SimilarityResult struct {
    Groups    []SimilarGroup    `json:"similar_groups"`
    Summary   DetectionSummary  `json:"summary"`
    Metadata  ResultMetadata    `json:"metadata"`
}

type SimilarGroup struct {
    ID                string             `json:"id"`
    SimilarityScore   float64            `json:"similarity_score"`
    Functions         []*FunctionInfo    `json:"functions"`
    RefactorSuggestion string            `json:"refactor_suggestion"`
}

type FunctionInfo struct {
    File      string `json:"file"`
    Function  string `json:"function"`
    StartLine int    `json:"start_line"`
    EndLine   int    `json:"end_line"`
    Hash      string `json:"hash"`
}

func (d *Detector) DetectSimilarity(functions []*ast.Function) (*SimilarityResult, error) {
    // 1. Pre-filter by hash for exact matches
    hashGroups := d.groupByHash(functions)
    
    // 2. Detailed comparison for potential matches
    similarGroups := []SimilarGroup{}
    
    for _, group := range hashGroups {
        if len(group) > 1 {
            // Exact hash match - high similarity
            sg := d.createSimilarGroup(group, 1.0)
            similarGroups = append(similarGroups, sg)
        }
    }
    
    // 3. Cross-hash comparison for partial similarities
    partialGroups := d.findPartialSimilarities(functions)
    similarGroups = append(similarGroups, partialGroups...)
    
    summary := d.generateSummary(functions, similarGroups)
    
    return &SimilarityResult{
        Groups:   similarGroups,
        Summary:  summary,
        Metadata: d.generateMetadata(),
    }, nil
}
```

#### 3.2 比較アルゴリズム実装

```go
// internal/similarity/algorithm.go
package similarity

import (
    "github.com/paveg/similarity-go/internal/ast"
)

type ComparisonAlgorithm interface {
    Compare(f1, f2 *ast.Function) (float64, error)
    BatchCompare(functions []*ast.Function) ([]SimilarGroup, error)
}

type StructuralComparison struct {
    weightAST     float64
    weightTokens  float64
    weightFlow    float64
    weightSignature float64
}

func NewStructuralComparison() *StructuralComparison {
    return &StructuralComparison{
        weightAST:       0.4,
        weightTokens:    0.3,
        weightFlow:      0.2,
        weightSignature: 0.1,
    }
}

func (sc *StructuralComparison) Compare(f1, f2 *ast.Function) (float64, error) {
    // 1. AST structure similarity
    astSim, err := sc.compareASTStructure(f1, f2)
    if err != nil {
        return 0, err
    }
    
    // 2. Token sequence similarity
    tokenSim := sc.compareTokenSequence(f1, f2)
    
    // 3. Control flow similarity
    flowSim := sc.compareControlFlow(f1, f2)
    
    // 4. Function signature similarity
    sigSim := sc.compareFunctionSignature(f1, f2)
    
    // Weighted average
    similarity := astSim*sc.weightAST + 
                 tokenSim*sc.weightTokens + 
                 flowSim*sc.weightFlow + 
                 sigSim*sc.weightSignature
    
    return similarity, nil
}

func (sc *StructuralComparison) compareASTStructure(f1, f2 *ast.Function) (float64, error) {
    // Tree edit distance based comparison
    return sc.treeEditDistance(f1.AST, f2.AST), nil
}

func (sc *StructuralComparison) compareTokenSequence(f1, f2 *ast.Function) float64 {
    tokens1 := sc.extractTokens(f1.AST)
    tokens2 := sc.extractTokens(f2.AST)
    
    return sc.jaccardSimilarity(tokens1, tokens2)
}

func (sc *StructuralComparison) jaccardSimilarity(set1, set2 []string) float64 {
    intersection := 0
    union := make(map[string]bool)
    
    set1Map := make(map[string]bool)
    for _, item := range set1 {
        set1Map[item] = true
        union[item] = true
    }
    
    for _, item := range set2 {
        union[item] = true
        if set1Map[item] {
            intersection++
        }
    }
    
    if len(union) == 0 {
        return 0
    }
    
    return float64(intersection) / float64(len(union))
}
```

### Phase 4: 並列処理・キャッシュ実装

#### 4.1 ワーカープール実装

```go
// internal/worker/pool.go
package worker

import (
    "context"
    "runtime"
    "sync"
)

type Pool struct {
    workerCount int
    jobQueue    chan Job
    resultQueue chan Result
    wg          sync.WaitGroup
    ctx         context.Context
    cancel      context.CancelFunc
}

type Job struct {
    ID   string
    Type JobType
    Data interface{}
}

type Result struct {
    JobID string
    Data  interface{}
    Error error
}

type JobType int

const (
    ParseFileJob JobType = iota
    CompareJob
    HashJob
)

func NewPool(workerCount int) *Pool {
    if workerCount <= 0 {
        workerCount = runtime.NumCPU()
    }
    
    ctx, cancel := context.WithCancel(context.Background())
    
    return &Pool{
        workerCount: workerCount,
        jobQueue:    make(chan Job, workerCount*2),
        resultQueue: make(chan Result, workerCount*2),
        ctx:         ctx,
        cancel:      cancel,
    }
}

func (p *Pool) Start() {
    for i := 0; i < p.workerCount; i++ {
        p.wg.Add(1)
        go p.worker(i)
    }
}

func (p *Pool) Stop() {
    close(p.jobQueue)
    p.cancel()
    p.wg.Wait()
    close(p.resultQueue)
}

func (p *Pool) Submit(job Job) {
    select {
    case p.jobQueue <- job:
    case <-p.ctx.Done():
    }
}

func (p *Pool) Results() <-chan Result {
    return p.resultQueue
}
```

#### 4.2 キャッシュシステム実装

```go
// internal/cache/manager.go
package cache

import (
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"
)

type Manager struct {
    cacheDir string
    ttl      time.Duration
    enabled  bool
}

func NewManager(cacheDir string, ttl time.Duration) *Manager {
    return &Manager{
        cacheDir: cacheDir,
        ttl:      ttl,
        enabled:  true,
    }
}

type CacheEntry struct {
    FileHash     string                `json:"file_hash"`
    Functions    []*CachedFunction     `json:"functions"`
    LastModified time.Time             `json:"last_modified"`
    CreatedAt    time.Time             `json:"created_at"`
}

type CachedFunction struct {
    Name      string `json:"name"`
    Hash      string `json:"hash"`
    StartLine int    `json:"start_line"`
    EndLine   int    `json:"end_line"`
    LineCount int    `json:"line_count"`
}

func (m *Manager) Get(filepath string) (*CacheEntry, error) {
    if !m.enabled {
        return nil, ErrCacheDisabled
    }
    
    cacheKey := m.generateCacheKey(filepath)
    cacheFile := filepath.Join(m.cacheDir, cacheKey+".json")
    
    data, err := os.ReadFile(cacheFile)
    if err != nil {
        return nil, err
    }
    
    var entry CacheEntry
    if err := json.Unmarshal(data, &entry); err != nil {
        return nil, err
    }
    
    // Check TTL
    if time.Since(entry.CreatedAt) > m.ttl {
        os.Remove(cacheFile)
        return nil, ErrCacheExpired
    }
    
    return &entry, nil
}

func (m *Manager) Set(filepath string, entry *CacheEntry) error {
    if !m.enabled {
        return nil
    }
    
    if err := os.MkdirAll(m.cacheDir, 0755); err != nil {
        return err
    }
    
    entry.CreatedAt = time.Now()
    cacheKey := m.generateCacheKey(filepath)
    cacheFile := filepath.Join(m.cacheDir, cacheKey+".json")
    
    data, err := json.MarshalIndent(entry, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile(cacheFile, data, 0644)
}

func (m *Manager) generateCacheKey(filepath string) string {
    hash := sha256.Sum256([]byte(filepath))
    return fmt.Sprintf("%x", hash)
}
```

この実装ガイドにより、段階的にツールを構築できる具体的な計画が整いました。各フェーズは独立してテスト可能で、incrementalな開発が可能です。
