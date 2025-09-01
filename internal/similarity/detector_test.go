package similarity

import (
	goast "go/ast"
	"go/token"
	"strings"
	"testing"

	"github.com/paveg/similarity-go/internal/ast"
	"github.com/paveg/similarity-go/internal/config"
	"github.com/paveg/similarity-go/internal/testhelpers"
)

func TestDetector_CalculateSimilarity(t *testing.T) {
	tests := []struct {
		name      string
		source1   string
		source2   string
		expected  float64
		threshold float64
	}{
		{
			name: "identical functions",
			source1: `package main
func add(a, b int) int {
	return a + b
}`,
			source2: `package main
func add(a, b int) int {
	return a + b
}`,
			expected:  1.0,
			threshold: 0.8,
		},
		{
			name: "same structure different variable names",
			source1: `package main
func add(a, b int) int {
	return a + b
}`,
			source2: `package main
func add(x, y int) int {
	return x + y
}`,
			expected:  1.0, // After normalization should be identical
			threshold: 0.8,
		},
		{
			name: "different functions",
			source1: `package main
func add(a, b int) int {
	return a + b
}`,
			source2: `package main
func multiply(a, b int) int {
	return a * b
}`,
			expected:  1.0, // Both normalized to same structure after variable renaming
			threshold: 0.8,
		},
		{
			name: "completely different functions",
			source1: `package main
func add(a, b int) int {
	return a + b
}`,
			source2: `package main
func processData(data []string) map[string]int {
	result := make(map[string]int)
	for _, item := range data {
		result[item] = len(item)
	}
	return result
}`,
			expected:  0.51, // Enhanced algorithm detects some token and structural similarity
			threshold: 0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDetector(tt.threshold)

			func1 := testhelpers.CreateFunctionFromSource(t, tt.source1, "add")
			func2 := testhelpers.CreateFunctionFromSource(t, tt.source2, "add")

			// Handle case where second function might have different name
			if func2 == nil && tt.name == "different functions" {
				func2 = testhelpers.CreateFunctionFromSource(t, tt.source2, "multiply")
			} else if func2 == nil && tt.name == "completely different functions" {
				func2 = testhelpers.CreateFunctionFromSource(t, tt.source2, "processData")
			}

			if func1 == nil || func2 == nil {
				t.Fatal("Failed to create test functions")
			}

			similarity := detector.CalculateSimilarity(func1, func2)

			// Allow some tolerance for floating point comparison
			tolerance := 0.1
			if testhelpers.AbsFloat(similarity-tt.expected) > tolerance {
				t.Errorf("Expected similarity %.2f, got %.2f", tt.expected, similarity)
			}
		})
	}
}

func TestDetector_IsAboveThreshold(t *testing.T) {
	tests := []struct {
		name       string
		threshold  float64
		similarity float64
		expected   bool
	}{
		{
			name:       "above threshold",
			threshold:  0.8,
			similarity: 0.8,
			expected:   true,
		},
		{
			name:       "equal to threshold",
			threshold:  0.8,
			similarity: 0.8,
			expected:   true,
		},
		{
			name:       "below threshold",
			threshold:  0.8,
			similarity: 0.6,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDetector(tt.threshold)
			result := detector.IsAboveThreshold(tt.similarity)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDetector_FindSimilarFunctions(t *testing.T) {
	functions := []*ast.Function{
		testhelpers.CreateFunctionFromSource(t, `package main
func add1(a, b int) int { return a + b }`, "add1"),
		testhelpers.CreateFunctionFromSource(t, `package main
func add2(x, y int) int { return x + y }`, "add2"),
		testhelpers.CreateFunctionFromSource(t, `package main
func multiply(a, b int) int { return a * b }`, "multiply"),
	}

	detector := NewDetector(0.8)
	matches := detector.FindSimilarFunctions(functions)

	// Should find add1 and add2 as similar
	if len(matches) == 0 {
		t.Error("Expected to find similar functions, got none")
	}

	// Verify the match contains the expected functions
	found := false

	for _, match := range matches {
		if (match.Function1.Name == "add1" && match.Function2.Name == "add2") ||
			(match.Function1.Name == "add2" && match.Function2.Name == "add1") {
			found = true

			if match.Similarity < 0.8 {
				t.Errorf("Expected similarity >= 0.8, got %.2f", match.Similarity)
			}
		}
	}

	if !found {
		t.Error("Expected to find match between add1 and add2")
	}
}

func TestDetector_EdgeCases(t *testing.T) {
	detector := NewDetector(0.8)

	tests := []struct {
		name     string
		func1    *ast.Function
		func2    *ast.Function
		expected float64
	}{
		{
			name:     "nil functions",
			func1:    nil,
			func2:    nil,
			expected: 0.0,
		},
		{
			name:     "one nil function",
			func1:    testhelpers.CreateFunctionFromSource(t, "package main\nfunc test() {}", "test"),
			func2:    nil,
			expected: 0.0,
		},
		{
			name:     "functions with nil AST",
			func1:    &ast.Function{Name: "test1", AST: nil},
			func2:    &ast.Function{Name: "test2", AST: nil},
			expected: 0.45, // Enhanced algorithm gives signature similarity component
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := detector.CalculateSimilarity(tt.func1, tt.func2)
			tolerance := 0.01
			if testhelpers.AbsFloat(similarity-tt.expected) > tolerance {
				t.Errorf("Expected similarity %.2f, got %.2f", tt.expected, similarity)
			}
		})
	}
}

func TestDetector_StringSimilarity(t *testing.T) {
	detector := NewDetector(0.5)

	tests := []struct {
		name     string
		s1       string
		s2       string
		expected float64
	}{
		{
			name:     "identical strings",
			s1:       "hello",
			s2:       "hello",
			expected: 1.0,
		},
		{
			name:     "empty strings",
			s1:       "",
			s2:       "",
			expected: 1.0,
		},
		{
			name:     "one empty string",
			s1:       "hello",
			s2:       "",
			expected: 0.0,
		},
		{
			name:     "similar length strings",
			s1:       "hello",
			s2:       "world",
			expected: 1.0, // Same length = perfect similarity in our simple implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := detector.stringSimilarity(tt.s1, tt.s2)
			if similarity != tt.expected {
				t.Errorf("Expected similarity %.2f, got %.2f", tt.expected, similarity)
			}
		})
	}
}

func TestDetector_TypeConversion(t *testing.T) {
	detector := NewDetector(0.5)

	// Test different AST types
	source := `package main
import "fmt"
func test(ptr *int, sel fmt.Stringer) string {
	return "test"
}`

	fn := testhelpers.CreateFunctionFromSource(t, source, "test")
	if fn == nil {
		t.Fatal("Failed to create function")
	}

	// Test getStructuralSignature with different types
	signature := detector.getStructuralSignature(fn)
	if signature == "" {
		t.Error("Expected non-empty structural signature")
	}
}

func TestDetector_BinaryExpressions(t *testing.T) {
	detector := NewDetector(0.5)

	tests := []struct {
		name     string
		source   string
		expected bool
	}{
		{
			name: "function with binary expression",
			source: `package main
func add(a, b int) int {
	return a + b
}`,
			expected: true,
		},
		{
			name: "function without binary expression",
			source: `package main
func hello() string {
	return "hello"
}`,
			expected: false,
		},
		{
			name:     "function with nil body",
			source:   "", // Will result in nil function
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.source == "" {
				// Test nil body case
				result := detector.hasBinaryExpressions(nil)
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}

				return
			}

			// Determine function name based on test case
			var funcName string
			switch tt.name {
			case "function with binary expression":
				funcName = "add"
			case "function without binary expression":
				funcName = "hello"
			default:
				funcName = "add" // fallback
			}

			fn := testhelpers.CreateFunctionFromSource(t, tt.source, funcName)

			if fn == nil {
				t.Fatal("Failed to create function")
			}

			result := detector.hasBinaryExpressions(fn.AST.Body)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDetector_CouldBeSimilar(t *testing.T) {
	detector := NewDetector(0.5)

	// Create functions with different characteristics
	shortFunc := &ast.Function{Name: "short", LineCount: 5}
	longFunc := &ast.Function{Name: "long", LineCount: 50}
	nilFunc := &ast.Function{Name: "nil", LineCount: 0}

	tests := []struct {
		name     string
		func1    *ast.Function
		func2    *ast.Function
		expected bool
	}{
		{
			name:     "similar sized functions",
			func1:    &ast.Function{Name: "test1", LineCount: 10},
			func2:    &ast.Function{Name: "test2", LineCount: 12},
			expected: true,
		},
		{
			name:     "very different sized functions",
			func1:    shortFunc,
			func2:    longFunc,
			expected: false,
		},
		{
			name:     "nil line count functions",
			func1:    nilFunc,
			func2:    &ast.Function{Name: "test", LineCount: 5},
			expected: true, // Can't determine, let full comparison decide
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.couldBeSimilar(tt.func1, tt.func2)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDetector_GetCacheKey(t *testing.T) {
	detector := NewDetector(0.5)

	hash1 := "abc123"
	hash2 := "def456"

	// Test that cache key is consistent regardless of order
	key1 := detector.getCacheKey(hash1, hash2)
	key2 := detector.getCacheKey(hash2, hash1)

	if key1 != key2 {
		t.Errorf("Expected consistent cache keys, got %s and %s", key1, key2)
	}

	// Test that different hashes produce different keys
	hash3 := "ghi789"
	key3 := detector.getCacheKey(hash1, hash3)

	if key1 == key3 {
		t.Error("Expected different cache keys for different hash combinations")
	}
}

func TestDetector_CompareNormalizedAST(t *testing.T) {
	detector := NewDetector(0.5)

	// Create test functions
	source1 := `package main
func add(x, y int) int {
	return x + y
}`

	source2 := `package main
func add(a, b int) int {
	return a + b
}`

	func1 := testhelpers.CreateFunctionFromSource(t, source1, "add")
	func2 := testhelpers.CreateFunctionFromSource(t, source2, "add")

	// These should be considered identical after normalization
	result := detector.compareNormalizedAST(func1, func2)
	if !result {
		t.Error("Expected normalized ASTs to be identical")
	}

	// Test with nil functions
	nilFunc := &ast.Function{Name: "nil", AST: nil}
	result = detector.compareNormalizedAST(func1, nilFunc)
	if result {
		t.Error("Expected false when comparing with nil AST")
	}
}

func TestDetector_CalculateTreeEditSimilarity(t *testing.T) {
	detector := NewDetector(0.5)

	// Test with identical functions
	source := `package main
func test() int {
	return 42
}`

	func1 := testhelpers.CreateFunctionFromSource(t, source, "test")
	func2 := testhelpers.CreateFunctionFromSource(t, source, "test")

	similarity := detector.calculateTreeEditSimilarity(func1, func2)
	if similarity < 0.9 {
		t.Errorf("Expected high similarity for identical functions, got %.2f", similarity)
	}

	// Test with nil functions
	similarity = detector.calculateTreeEditSimilarity(nil, func1)
	if similarity != 0.0 {
		t.Errorf("Expected 0.0 for nil function, got %.2f", similarity)
	}
}

func TestDetector_CountASTNodes(t *testing.T) {
	detector := NewDetector(0.5)

	// Test with nil node
	count := detector.countASTNodes(nil)
	if count != 0 {
		t.Errorf("Expected 0 for nil node, got %d", count)
	}

	// Test with simple function
	source := `package main
func simple() {
	x := 1
	return
}`

	fn := testhelpers.CreateFunctionFromSource(t, source, "simple")
	count = detector.countASTNodes(fn.AST)
	if count <= 0 {
		t.Errorf("Expected positive count for function AST, got %d", count)
	}
}

// TestDetector_CountASTNodesComprehensive tests all AST node types in countASTNodes.
func TestDetector_CountASTNodesComprehensive(t *testing.T) {
	detector := NewDetector(0.5)

	tests := []struct {
		name     string
		source   string
		minCount int
		desc     string
	}{
		{
			name: "simple assignment",
			source: `package main
func test() {
	x := 42
}`,
			minCount: 3, // FuncDecl + BlockStmt + AssignStmt + ...
			desc:     "simple assignment should count multiple nodes",
		},
		{
			name: "binary expression",
			source: `package main
func test() {
	result := a + b
}`,
			minCount: 5, // FuncDecl + BlockStmt + AssignStmt + BinaryExpr + operands
			desc:     "binary expressions should count operands",
		},
		{
			name: "unary expression",
			source: `package main
func test() {
	result := !flag
}`,
			minCount: 4, // FuncDecl + BlockStmt + AssignStmt + UnaryExpr + operand
			desc:     "unary expressions should count operand",
		},
		{
			name: "function call",
			source: `package main
func test() {
	result := myFunc(a, b, c)
}`,
			minCount: 7, // FuncDecl + BlockStmt + AssignStmt + CallExpr + Fun + 3 Args
			desc:     "function calls should count function and all arguments",
		},
		{
			name: "return statement",
			source: `package main
func test() int {
	return a + b
}`,
			minCount: 5, // FuncDecl + BlockStmt + ReturnStmt + BinaryExpr + operands
			desc:     "return statements should count return values",
		},
		{
			name: "multiple assignment",
			source: `package main
func test() {
	x, y := a + b, c * d
}`,
			minCount: 9, // FuncDecl + BlockStmt + AssignStmt + 2 lhs + 2 rhs (each with ops)
			desc:     "multiple assignments should count all sides",
		},
		{
			name: "expression statement",
			source: `package main
func test() {
	fmt.Println("hello")
}`,
			minCount: 4, // FuncDecl + BlockStmt + ExprStmt + CallExpr + ...
			desc:     "expression statements should count the expression",
		},
		{
			name: "if statement with init",
			source: `package main
func test() {
	if x := getValue(); x > 0 {
		doSomething()
	}
}`,
			minCount: 8, // FuncDecl + BlockStmt + IfStmt + Init + Cond + Body + ...
			desc:     "if statements should count init, condition, and body",
		},
		{
			name: "if statement with else",
			source: `package main
func test() {
	if condition {
		doThis()
	} else {
		doThat()
	}
}`,
			minCount: 7, // FuncDecl + BlockStmt + IfStmt + Cond + Body + Else + ...
			desc:     "if statements with else should count else branch",
		},
		{
			name: "for loop with all components",
			source: `package main
func test() {
	for i := 0; i < 10; i++ {
		doWork()
	}
}`,
			minCount: 10, // FuncDecl + BlockStmt + ForStmt + Init + Cond + Post + Body + ...
			desc:     "for loops should count init, condition, post, and body",
		},
		{
			name: "nested structures",
			source: `package main
func test() {
	if x > 0 {
		for i := 0; i < x; i++ {
			if i%2 == 0 {
				result += i
			}
		}
	}
}`,
			minCount: 15, // Deeply nested structures should have high node count
			desc:     "nested structures should count all nested nodes",
		},
		{
			name: "complex expression",
			source: `package main
func test() {
	result := ((a + b) * c) / (d - e) + f
}`,
			minCount: 8, // Complex nested binary operations
			desc:     "complex expressions should count all sub-expressions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			function := testhelpers.CreateFunctionFromSource(t, tt.source, "test")
			if function == nil || function.AST == nil {
				t.Fatal("Failed to create function from source")
			}

			count := detector.countASTNodes(function.AST)
			if count < tt.minCount {
				t.Errorf("%s: expected at least %d nodes, got %d",
					tt.desc, tt.minCount, count)
			}

			// Count should be positive for any valid AST
			if count <= 0 {
				t.Errorf("Node count should be positive, got %d", count)
			}
		})
	}
}

// TestDetector_CountASTNodesEdgeCases tests edge cases and unsupported node types.
func TestDetector_CountASTNodesEdgeCases(t *testing.T) {
	detector := NewDetector(0.5)

	tests := []struct {
		name     string
		setup    func() goast.Node
		expected int
		desc     string
	}{
		{
			name: "nil node",
			setup: func() goast.Node {
				return nil
			},
			expected: 0,
			desc:     "nil nodes should return 0",
		},
		{
			name: "unsupported node type",
			setup: func() goast.Node {
				return &goast.Ident{Name: "test"}
			},
			expected: 1,
			desc:     "unsupported node types should return 1 (just the node itself)",
		},
		{
			name: "empty block statement",
			setup: func() goast.Node {
				return &goast.BlockStmt{List: []goast.Stmt{}}
			},
			expected: 1,
			desc:     "empty block statements should return 1",
		},
		{
			name: "function with nil body",
			setup: func() goast.Node {
				return &goast.FuncDecl{
					Name: &goast.Ident{Name: "test"},
					Type: &goast.FuncType{},
					Body: nil, // nil body
				}
			},
			expected: 2,
			desc:     "function with nil body should handle gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := tt.setup()
			count := detector.countASTNodes(node)
			if count != tt.expected {
				t.Errorf("%s: expected %d, got %d", tt.desc, tt.expected, count)
			}
		})
	}
}

// TestDetector_CountASTNodesConsistency tests that counting is consistent.
func TestDetector_CountASTNodesConsistency(t *testing.T) {
	detector := NewDetector(0.5)

	source := `package main
func test() {
	x := a + b
	if x > 0 {
		return x * 2
	}
	return 0
}`

	function := testhelpers.CreateFunctionFromSource(t, source, "test")
	if function == nil || function.AST == nil {
		t.Fatal("Failed to create function")
	}

	// Count multiple times to ensure consistency
	count1 := detector.countASTNodes(function.AST)
	count2 := detector.countASTNodes(function.AST)
	count3 := detector.countASTNodes(function.AST)

	if count1 != count2 || count2 != count3 {
		t.Errorf("Node counting should be consistent: %d, %d, %d", count1, count2, count3)
	}

	// Count should be reasonable for this structure
	if count1 < 5 || count1 > 50 {
		t.Errorf("Node count %d seems unreasonable for this function", count1)
	}
}

func TestDetector_CalculateStructuralSimilarity(t *testing.T) {
	detector := NewDetector(0.5)

	// Test with functions that have same signature
	source1 := `package main
func test(a int) int {
	return a + 1
}`

	source2 := `package main
func test(a int) int {
	return a * 2
}`

	func1 := testhelpers.CreateFunctionFromSource(t, source1, "test")
	func2 := testhelpers.CreateFunctionFromSource(t, source2, "test")

	similarity := detector.calculateStructuralSimilarity(func1, func2)
	if similarity <= 0 {
		t.Errorf("Expected positive similarity for same signature functions, got %.2f", similarity)
	}

	// Test with nil AST
	nilFunc := &ast.Function{Name: "nil", AST: nil}
	similarity = detector.calculateStructuralSimilarity(func1, nilFunc)
	if similarity != 0.0 {
		t.Errorf("Expected 0.0 for nil AST, got %.2f", similarity)
	}
}

func TestDetector_CalculateSignatureSimilarity(t *testing.T) {
	detector := NewDetector(0.5)

	// Test with identical signatures
	func1 := &ast.Function{Name: "test1"}
	func2 := &ast.Function{Name: "test2"}

	// Mock GetSignature to return predictable values
	func1.AST = &goast.FuncDecl{
		Type: &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Type: &goast.Ident{Name: "int"}},
				},
			},
		},
	}
	func2.AST = &goast.FuncDecl{
		Type: &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Type: &goast.Ident{Name: "int"}},
				},
			},
		},
	}

	similarity := detector.calculateSignatureSimilarity(func1, func2)
	if similarity != 1.0 {
		t.Errorf("Expected 1.0 for identical signatures, got %.2f", similarity)
	}
}

func TestDetector_GetStructuralSignature(t *testing.T) {
	detector := NewDetector(0.5)

	// Test with nil AST
	nilFunc := &ast.Function{Name: "nil", AST: nil}
	sig := detector.getStructuralSignature(nilFunc)
	if sig != "" {
		t.Errorf("Expected empty signature for nil AST, got %s", sig)
	}

	// Test with function
	source := `package main
func test(a int, b string) (int, error) {
	return 0, nil
}`

	fn := testhelpers.CreateFunctionFromSource(t, source, "test")
	sig = detector.getStructuralSignature(fn)
	if sig == "" {
		t.Error("Expected non-empty signature for valid function")
	}
	if !strings.Contains(sig, "func(") {
		t.Errorf("Expected signature to contain 'func(', got %s", sig)
	}
}

func TestDetector_CompareBodyStructure(t *testing.T) {
	detector := NewDetector(0.5)

	// Test with nil bodies
	similarity := detector.compareBodyStructure(nil, nil)
	if similarity != 1.0 {
		t.Errorf("Expected 1.0 for both nil bodies, got %.2f", similarity)
	}

	// Test with one nil body
	source := `package main
func test() {
	x := 1
}`

	fn := testhelpers.CreateFunctionFromSource(t, source, "test")
	similarity = detector.compareBodyStructure(fn.AST.Body, nil)
	if similarity != 0.0 {
		t.Errorf("Expected 0.0 for nil body comparison, got %.2f", similarity)
	}
}

func TestDetector_StatementsStructurallyEqual(t *testing.T) {
	detector := NewDetector(0.5)

	// Create different statement types
	returnStmt1 := &goast.ReturnStmt{}
	returnStmt2 := &goast.ReturnStmt{}
	assignStmt := &goast.AssignStmt{}

	// Same types should be equal
	if !detector.statementsStructurallyEqual(returnStmt1, returnStmt2) {
		t.Error("Expected same statement types to be structurally equal")
	}

	// Different types should not be equal
	if detector.statementsStructurallyEqual(returnStmt1, assignStmt) {
		t.Error("Expected different statement types to not be structurally equal")
	}
}

func TestDetector_GenerateASTHash(t *testing.T) {
	detector := NewDetector(0.5)

	// Test with nil node
	hash := detector.generateASTHash(nil)
	if hash != "" {
		t.Errorf("Expected empty hash for nil node, got %s", hash)
	}

	// Test with valid AST node
	source := `package main
func test() {
	return
}`

	fn := testhelpers.CreateFunctionFromSource(t, source, "test")
	hash = detector.generateASTHash(fn.AST)
	if hash == "" {
		t.Error("Expected non-empty hash for valid AST")
	}
}

func TestDetector_TypeToString(t *testing.T) {
	detector := NewDetector(0.5)

	tests := []struct {
		name     string
		expr     goast.Expr
		expected string
	}{
		{
			name:     "simple identifier",
			expr:     &goast.Ident{Name: "int"},
			expected: "int",
		},
		{
			name:     "pointer type",
			expr:     &goast.StarExpr{X: &goast.Ident{Name: "string"}},
			expected: "*string",
		},
		{
			name:     "selector expression",
			expr:     &goast.SelectorExpr{X: &goast.Ident{Name: "fmt"}, Sel: &goast.Ident{Name: "Print"}},
			expected: "fmt.Print",
		},
		{
			name:     "unknown type",
			expr:     &goast.BasicLit{Kind: token.INT, Value: "42"},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.typeToString(tt.expr)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestDetector_HasSimilarOperations(t *testing.T) {
	detector := NewDetector(0.5)

	// Test with functions that have binary expressions
	source1 := `package main
func add(a, b int) int {
	return a + b
}`

	source2 := `package main
func multiply(a, b int) int {
	return a * b
}`

	func1 := testhelpers.CreateFunctionFromSource(t, source1, "add")
	func2 := testhelpers.CreateFunctionFromSource(t, source2, "multiply")

	result := detector.hasSimilarOperations(func1, func2)
	if !result {
		t.Error("Expected functions with binary operations to be considered similar")
	}

	// Test with function without binary expression
	source3 := `package main
func hello() string {
	return "hello"
}`

	func3 := testhelpers.CreateFunctionFromSource(t, source3, "hello")
	result = detector.hasSimilarOperations(func1, func3)
	if result {
		t.Error("Expected functions with different operation types to not be similar")
	}
}

func TestNewDetectorWithConfig(t *testing.T) {
	cfg := config.Default()
	threshold := 0.8
	detector := NewDetectorWithConfig(threshold, cfg)

	if detector.threshold != threshold {
		t.Errorf("expected threshold %f, got %f", threshold, detector.threshold)
	}

	if detector.config != cfg {
		t.Error("expected detector to use provided config")
	}

	// Test with nil config should not panic
	detector2 := NewDetectorWithConfig(threshold, nil)
	if detector2 == nil {
		t.Error("expected detector to be created even with nil config")
	}
}

func TestFindSimilarFunctionsWithProcessor(t *testing.T) {
	detector := NewDetector(0.8)

	source1 := `package main
func add(a, b int) int {
	return a + b
}`

	source2 := `package main
func sum(x, y int) int {
	return x + y
}`

	func1 := testhelpers.CreateFunctionFromSource(t, source1, "add")
	func2 := testhelpers.CreateFunctionFromSource(t, source2, "sum")

	functions := []*ast.Function{func1, func2}

	// Create a mock processor
	mockProcessor := &MockProcessor{detector: detector}

	matches, err := detector.FindSimilarFunctionsWithProcessor(mockProcessor, functions, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if matches == nil {
		t.Error("expected matches to be returned")
	}

	if !mockProcessor.called {
		t.Error("expected processor to be called")
	}
}

// MockProcessor implements ParallelProcessor for testing.
type MockProcessor struct {
	detector *Detector
	called   bool
}

func (m *MockProcessor) FindSimilarFunctions(functions []*ast.Function, _ func(completed, total int)) ([]Match, error) {
	m.called = true
	return m.detector.FindSimilarFunctions(functions), nil
}

func TestTreeEditDistanceNilCases(t *testing.T) {
	// Test with nil ASTs
	distance := TreeEditDistance(nil, nil)
	if distance != 0 {
		t.Errorf("expected distance 0 for nil ASTs, got %d", distance)
	}

	// Test with one nil AST
	source1 := `package main
func test() int {
	return 1
}`
	func1 := testhelpers.CreateFunctionFromSource(t, source1, "test")

	distance = TreeEditDistance(func1.AST, nil)
	if distance <= 0 {
		t.Error("expected positive distance when comparing AST to nil")
	}

	distance = TreeEditDistance(nil, func1.AST)
	if distance <= 0 {
		t.Error("expected positive distance when comparing nil to AST")
	}
}

func TestGetNodeChildrenAndType(t *testing.T) {
	source := `package main
func test() int {
	if true {
		return 1
	}
	return 0
}`
	function := testhelpers.CreateFunctionFromSource(t, source, "test")

	// Test with function declaration
	nodeType := getNodeType(function.AST)
	if nodeType == "" {
		t.Error("expected non-empty node type for function declaration")
	}

	children := getNodeChildren(function.AST)
	// Function declarations may or may not have children depending on structure
	if children == nil {
		t.Log("Function declaration has no children, which is acceptable")
	}

	// Test with body statements
	if function.AST.Body != nil && len(function.AST.Body.List) > 0 {
		stmt := function.AST.Body.List[0]
		stmtType := getNodeType(stmt)
		if stmtType == "" {
			t.Error("expected non-empty node type for statement")
		}

		stmtChildren := getNodeChildren(stmt)
		// Some statements may not have children, that's ok
		if stmtChildren == nil {
			t.Log("Statement has no children, which is acceptable")
		}
	}
}

func TestCalculateChildrenDistanceAlgorithm(t *testing.T) {
	source1 := `package main
func test1() int {
	if true {
		return 1
	}
	return 0
}`

	source2 := `package main
func test2() int {
	if false {
		return 2
	}
	return 0
}`

	func1 := testhelpers.CreateFunctionFromSource(t, source1, "test1")
	func2 := testhelpers.CreateFunctionFromSource(t, source2, "test2")

	// Test calculateChildrenDistance with function AST nodes
	if func1.AST.Body != nil && func2.AST.Body != nil {
		distance := calculateChildrenDistance(func1.AST.Body, func2.AST.Body)
		if distance < 0 {
			t.Errorf("expected non-negative distance, got %d", distance)
		}
	}

	// Test with same nodes
	distance := calculateChildrenDistance(func1.AST, func1.AST)
	if distance != 0 {
		t.Errorf("expected distance 0 for same nodes, got %d", distance)
	}
}
