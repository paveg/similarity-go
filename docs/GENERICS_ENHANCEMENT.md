# Go Generics Enhancement Plan - Similarity Detection Tool

## Leveraging Generics for Design Improvements

A plan to utilize Go 1.18+ Generics to improve the existing design into a more type-safe and maintainable implementation.

## 1. Type-Safe Collections & Data Structures

### 1.1 Generic Result Type

```go
// Before (using interface{})
type Result struct {
    JobID string
    Data  interface{}
    Error error
}

// After (using Generics)
type Result[T any] struct {
    JobID string
    Data  T
    Error error
}

type ParseResult = Result[*ast.File]
type ComparisonResult = Result[float64]
type HashResult = Result[string]
```

### 1.2 Type-Safe Worker Pool

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

// Concrete type definitions
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

### 1.3 Generic Cache System

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

// Concrete cache instances
type FunctionCache = Cache[string, *CachedFunction]
type FileCache = Cache[string, *CacheEntry]
```

## 2. Generic Algorithms & Comparison Processing

### 2.1 Abstract Comparison Algorithms

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

// Generic structural comparison algorithm implementation
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
    // Generic comparison logic
    hashA, hashB := a.Hash(), b.Hash()
    if hashA == hashB {
        return 1.0, nil
    }
    
    normalizedA := a.Normalize()
    normalizedB := b.Normalize()
    
    return sc.computeSimilarity(normalizedA, normalizedB)
}
```

### 2.2 Function Type Constraint Definitions

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

// Comparable interface implementation
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

## 3. Generic Containers & Utilities

### 3.1 Generic Collection Operations

```go
// pkg/collections/generic.go
package collections

// Filtering
func Filter[T any](slice []T, predicate func(T) bool) []T {
    result := make([]T, 0, len(slice))
    for _, item := range slice {
        if predicate(item) {
            result = append(result, item)
        }
    }
    return result
}

// Mapping
func Map[T, U any](slice []T, mapper func(T) U) []U {
    result := make([]U, len(slice))
    for i, item := range slice {
        result[i] = mapper(item)
    }
    return result
}

// Grouping
func GroupBy[T any, K comparable](slice []T, keyFunc func(T) K) map[K][]T {
    result := make(map[K][]T)
    for _, item := range slice {
        key := keyFunc(item)
        result[key] = append(result[key], item)
    }
    return result
}

// Parallel mapping
func ParallelMap[T, U any](slice []T, mapper func(T) U, workers int) []U {
    if workers <= 0 {
        workers = runtime.NumCPU()
    }
    
    input := make(chan indexedItem[T], len(slice))
    output := make(chan indexedItem[U], len(slice))
    
    // Start workers
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
    
    // Send data
    go func() {
        defer close(input)
        for i, item := range slice {
            input <- indexedItem[T]{index: i, value: item}
        }
    }()
    
    // Collect results
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

### 3.2 Optional Type Implementation

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

## 4. Enhanced Error Handling

### 4.1 Result Type Pattern Implementation

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

### 4.2 Type-Safe Parser Implementation

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
    // Check cache
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

## 5. Type-Safe Configuration & Dependency Injection

### 5.1 Type-Safe Configuration

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

// Concrete configuration types
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

### 5.2 Dependency Injection Container

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

## 6. Implementation Benefits

### 6.1 Improved Type Safety

- **Compile-time Error Detection**: Reduce usage of `interface{}` and detect type mismatches at compile time
- **Enhanced Auto-completion**: Improved type inference and code completion accuracy in IDEs
- **Refactoring Safety**: Safe refactoring based on type information

### 6.2 Performance Improvements

- **Avoid Boxing**: Reduce heap allocations from `interface{}` boxing
- **Inline Optimization**: Inline expansion of generic functions for optimization
- **Memory Efficiency**: Efficient memory usage through type-specialized data structures

### 6.3 Enhanced Maintainability

- **Code Reusability**: Generic algorithms and data structures
- **Improved Readability**: Clear API design through type constraints
- **Testability**: Type-safe mocks and test helpers

## 7. Migration Plan

### Phase 1: Foundation Type Introduction

- [ ] Implement Result and Optional types
- [ ] Implement generic collection operations
- [ ] Prepare existing code for Generics compatibility

### Phase 2: Core Feature Type Safety

- [ ] Genericize Worker Pool
- [ ] Genericize Cache System
- [ ] Type-safe Parser implementation

### Phase 3: Algorithm Abstraction

- [ ] Genericize comparison algorithms
- [ ] Unify result types
- [ ] Type-safe configuration system

This enhancement leverages Go 1.18+ features to create a more robust, maintainable code similarity verification tool with improved type safety and performance characteristics.