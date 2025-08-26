# Go Generics Enhancement Plan - Similarity Detection Tool

## Genericsの活用機会と設計改善

Go 1.18以降のGenericsを活用して、既存設計をより型安全で保守性の高い実装に改善する計画です。

## 1. コレクション・データ構造の型安全化

### 1.1 Result型の汎用化

```go
// Before (interface{}を使用)
type Result struct {
    JobID string
    Data  interface{}
    Error error
}

// After (Genericsを使用)
type Result[T any] struct {
    JobID string
    Data  T
    Error error
}

type ParseResult = Result[*ast.File]
type ComparisonResult = Result[float64]
type HashResult = Result[string]
```

### 1.2 Worker Poolの型安全化

```go
// internal/worker/pool.go
type Pool[TJob any, TResult any] struct {
    workerCount int
    jobQueue    chan Job[TJob]
    resultQueue chan Result[TResult]
    processor   JobProcessor[TJob, TResult]
    wg          sync.WaitGroup
    ctx         context.Context
    cancel      context.CancelFunc
}

type Job[T any] struct {
    ID   string
    Type JobType
    Data T
}

type JobProcessor[TJob any, TResult any] interface {
    Process(job Job[TJob]) Result[TResult]
}

// 具体的な型定義
type FileParseJob struct {
    FilePath string
    Content  []byte
}

type ParsedFileResult struct {
    Functions []*Function
    Metadata  FileMetadata
}

type FileParserPool = Pool[FileParseJob, ParsedFileResult]
```

### 1.3 キャッシュシステムの汎用化

```go
// internal/cache/manager.go
type Cache[K comparable, V any] interface {
    Get(key K) (V, bool)
    Set(key K, value V) error
    Delete(key K) error
    Clear() error
    Keys() []K
}

type LRUCache[K comparable, V any] struct {
    capacity int
    items    map[K]*cacheItem[V]
    head     *cacheItem[V]
    tail     *cacheItem[V]
    mu       sync.RWMutex
}

type cacheItem[V any] struct {
    key   K
    value V
    prev  *cacheItem[V]
    next  *cacheItem[V]
}

// 具体的なキャッシュインスタンス
type FunctionCache = Cache[string, *CachedFunction]
type FileCache = Cache[string, *CacheEntry]
```

## 2. アルゴリズム・比較処理の汎用化

### 2.1 比較アルゴリズムの抽象化

```go
// internal/similarity/algorithm.go
type Comparable interface {
    Hash() string
    Normalize() Comparable
}

type Comparator[T Comparable] interface {
    Compare(a, b T) (float64, error)
    BatchCompare(items []T) ([]SimilarGroup[T], error)
}

type SimilarGroup[T Comparable] struct {
    ID              string    `json:"id"`
    SimilarityScore float64   `json:"similarity_score"`
    Items           []T       `json:"items"`
    RefactorSuggestion string `json:"refactor_suggestion"`
}

// 構造比較アルゴリズムの汎用実装
type StructuralComparator[T Comparable] struct {
    threshold float64
    weights   ComparisonWeights
}

func NewStructuralComparator[T Comparable](threshold float64) *StructuralComparator[T] {
    return &StructuralComparator[T]{
        threshold: threshold,
        weights: ComparisonWeights{
            Structure: 0.4,
            Semantic:  0.3,
            Syntax:    0.2,
            Context:   0.1,
        },
    }
}

func (sc *StructuralComparator[T]) Compare(a, b T) (float64, error) {
    // 汎用比較ロジック
    hashA, hashB := a.Hash(), b.Hash()
    if hashA == hashB {
        return 1.0, nil
    }
    
    normalizedA := a.Normalize()
    normalizedB := b.Normalize()
    
    return sc.computeSimilarity(normalizedA, normalizedB)
}
```

### 2.2 関数型の型制約定義

```go
// pkg/types/constraints.go
type ASTNode interface {
    ast.Node
    Pos() token.Pos
    End() token.Pos
}

type Function struct {
    Name       string
    File       string
    StartLine  int
    EndLine    int
    AST        *ast.FuncDecl
    Normalized *ast.FuncDecl
    hash       string
    signature  string
}

// Comparableインターフェースの実装
func (f *Function) Hash() string {
    if f.hash == "" {
        hasher := NewStructureHasher()
        f.hash, _ = hasher.HashFunction(f.AST)
    }
    return f.hash
}

func (f *Function) Normalize() Comparable {
    if f.Normalized == nil {
        normalizer := NewNormalizer()
        f.Normalized = normalizer.Normalize(f.AST)
    }
    
    return &Function{
        Name:       f.Name,
        File:       f.File,
        StartLine:  f.StartLine,
        EndLine:    f.EndLine,
        AST:        f.Normalized,
        Normalized: f.Normalized,
    }
}
```

## 3. コンテナ・ユーティリティの汎用化

### 3.1 汎用コレクション操作

```go
// pkg/collections/generic.go
package collections

// フィルタリング
func Filter[T any](slice []T, predicate func(T) bool) []T {
    result := make([]T, 0, len(slice))
    for _, item := range slice {
        if predicate(item) {
            result = append(result, item)
        }
    }
    return result
}

// マッピング
func Map[T, U any](slice []T, mapper func(T) U) []U {
    result := make([]U, len(slice))
    for i, item := range slice {
        result[i] = mapper(item)
    }
    return result
}

// グルーピング
func GroupBy[T any, K comparable](slice []T, keyFunc func(T) K) map[K][]T {
    result := make(map[K][]T)
    for _, item := range slice {
        key := keyFunc(item)
        result[key] = append(result[key], item)
    }
    return result
}

// 並列マッピング
func ParallelMap[T, U any](slice []T, mapper func(T) U, workers int) []U {
    if workers <= 0 {
        workers = runtime.NumCPU()
    }
    
    input := make(chan indexedItem[T], len(slice))
    output := make(chan indexedItem[U], len(slice))
    
    // Workers起動
    for i := 0; i < workers; i++ {
        go func() {
            for item := range input {
                output <- indexedItem[U]{
                    index: item.index,
                    value: mapper(item.value),
                }
            }
        }()
    }
    
    // データ送信
    go func() {
        defer close(input)
        for i, item := range slice {
            input <- indexedItem[T]{index: i, value: item}
        }
    }()
    
    // 結果収集
    result := make([]U, len(slice))
    for i := 0; i < len(slice); i++ {
        item := <-output
        result[item.index] = item.value
    }
    
    return result
}

type indexedItem[T any] struct {
    index int
    value T
}
```

### 3.2 Optional型の実装

```go
// pkg/types/optional.go
type Optional[T any] struct {
    value *T
}

func Some[T any](value T) Optional[T] {
    return Optional[T]{value: &value}
}

func None[T any]() Optional[T] {
    return Optional[T]{value: nil}
}

func (o Optional[T]) IsSome() bool {
    return o.value != nil
}

func (o Optional[T]) IsNone() bool {
    return o.value == nil
}

func (o Optional[T]) Unwrap() T {
    if o.value == nil {
        panic("called Unwrap on None value")
    }
    return *o.value
}

func (o Optional[T]) UnwrapOr(defaultValue T) T {
    if o.value == nil {
        return defaultValue
    }
    return *o.value
}

func (o Optional[T]) Map(mapper func(T) T) Optional[T] {
    if o.value == nil {
        return None[T]()
    }
    return Some(mapper(*o.value))
}
```

## 4. エラーハンドリングの改善

### 4.1 Result型パターンの実装

```go
// pkg/types/result.go
type Result[T any] struct {
    value *T
    error error
}

func Ok[T any](value T) Result[T] {
    return Result[T]{value: &value, error: nil}
}

func Err[T any](err error) Result[T] {
    return Result[T]{value: nil, error: err}
}

func (r Result[T]) IsOk() bool {
    return r.error == nil
}

func (r Result[T]) IsErr() bool {
    return r.error != nil
}

func (r Result[T]) Unwrap() T {
    if r.error != nil {
        panic(r.error)
    }
    return *r.value
}

func (r Result[T]) UnwrapOr(defaultValue T) T {
    if r.error != nil {
        return defaultValue
    }
    return *r.value
}

func (r Result[T]) Map[U any](mapper func(T) U) Result[U] {
    if r.error != nil {
        return Err[U](r.error)
    }
    return Ok(mapper(*r.value))
}

func (r Result[T]) AndThen[U any](f func(T) Result[U]) Result[U] {
    if r.error != nil {
        return Err[U](r.error)
    }
    return f(*r.value)
}
```

### 4.2 型安全なパーサー実装

```go
// internal/ast/parser.go
type Parser[T ASTNode] struct {
    fileSet *token.FileSet
    cache   Cache[string, Result[[]T]]
}

func NewParser[T ASTNode]() *Parser[T] {
    return &Parser[T]{
        fileSet: token.NewFileSet(),
        cache:   NewLRUCache[string, Result[[]T]](1000),
    }
}

func (p *Parser[T]) ParseFile(filename string) Result[[]T] {
    // キャッシュチェック
    if cached, exists := p.cache.Get(filename); exists {
        return cached
    }
    
    result := p.parseFileInternal(filename)
    p.cache.Set(filename, result)
    
    return result
}

func (p *Parser[*Function]) extractFunctions(file *ast.File, filename string) []*Function {
    var functions []*Function
    
    ast.Inspect(file, func(n ast.Node) bool {
        if funcDecl, ok := n.(*ast.FuncDecl); ok && funcDecl.Body != nil {
            fn := &Function{
                Name:      funcDecl.Name.Name,
                File:      filename,
                AST:       funcDecl,
                StartLine: p.fileSet.Position(funcDecl.Pos()).Line,
                EndLine:   p.fileSet.Position(funcDecl.End()).Line,
            }
            functions = append(functions, fn)
        }
        return true
    })
    
    return functions
}
```

## 5. 設定・依存性注入の型安全化

### 5.1 設定の型安全化

```go
// internal/config/typed.go
type Config[T any] struct {
    value T
}

func NewConfig[T any](value T) *Config[T] {
    return &Config[T]{value: value}
}

func (c *Config[T]) Get() T {
    return c.value
}

func (c *Config[T]) Update(updater func(T) T) {
    c.value = updater(c.value)
}

// 具体的な設定型
type SimilarityConfig struct {
    Threshold     float64
    MinLines      int
    Workers       int
    CacheEnabled  bool
    IgnoreFile    string
    OutputFormat  string
}

type AlgorithmConfig struct {
    Weights ComparisonWeights
    Method  string
}

type ComparisonWeights struct {
    Structure float64
    Semantic  float64
    Syntax    float64
    Context   float64
}
```

### 5.2 依存性注入コンテナ

```go
// pkg/di/container.go
type Container interface {
    Register[T any](factory func() T)
    RegisterSingleton[T any](factory func() T)
    Resolve[T any]() T
    ResolveOptional[T any]() Optional[T]
}

type container struct {
    factories  map[reflect.Type]func() interface{}
    singletons map[reflect.Type]interface{}
    mu         sync.RWMutex
}

func NewContainer() Container {
    return &container{
        factories:  make(map[reflect.Type]func() interface{}),
        singletons: make(map[reflect.Type]interface{}),
    }
}

func (c *container) Register[T any](factory func() T) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    var zero T
    typ := reflect.TypeOf(zero)
    c.factories[typ] = func() interface{} {
        return factory()
    }
}

func (c *container) Resolve[T any]() T {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    var zero T
    typ := reflect.TypeOf(zero)
    
    if singleton, exists := c.singletons[typ]; exists {
        return singleton.(T)
    }
    
    if factory, exists := c.factories[typ]; exists {
        return factory().(T)
    }
    
    panic(fmt.Sprintf("no factory registered for type %T", zero))
}
```

## 6. 実装における利点

### 6.1 型安全性の向上

- **コンパイル時エラー検出**: `interface{}`の使用を削減し、型不整合をコンパイル時に検出
- **自動補完の改善**: IDEでの型推論とコード補完の精度向上
- **リファクタリング安全性**: 型情報に基づいた安全なリファクタリング

### 6.2 パフォーマンス改善

- **ボクシング回避**: `interface{}`によるヒープアロケーションを削減
- **インライン最適化**: ジェネリック関数のインライン展開による最適化
- **メモリ効率**: 型特化されたデータ構造による効率的なメモリ使用

### 6.3 保守性の向上

- **コードの再利用**: 汎用的なアルゴリズムとデータ構造
- **可読性向上**: 型制約による明確なAPI設計
- **テスタビリティ**: 型安全なモックとテストヘルパー

## 7. マイグレーション計画

### Phase 1: 基盤型の導入

- [ ] Result型、Optional型の実装
- [ ] 汎用コレクション操作の実装
- [ ] 既存コードのGenerics対応準備

### Phase 2: コア機能の型安全化

- [ ] Worker Poolのジェネリック化
- [ ] キャッシュシステムのジェネリック化
- [ ] パーサーの型安全化

### Phase 3: アルゴリズムの抽象化

- [ ] 比較アルゴリズムのジェネリック化
- [ ] 結果型の統一
- [ ] 設定システムの型安全化

この改善により、Go 1.18+の最新機能を活用した、より堅牢で保守性の高いコード類似性検証ツールが実現できます。
