package similarity

import (
	"bytes"
	goast "go/ast"
	"go/format"
	"go/scanner"
	"go/token"
	"strings"

	"github.com/paveg/similarity-go/internal/ast"
	"github.com/paveg/similarity-go/pkg/mathutil"
)

// TreeEditDistance calculates the edit distance between two AST nodes.
// This implements a simplified tree edit distance algorithm.
func TreeEditDistance(node1, node2 goast.Node) int {
	if node1 == nil && node2 == nil {
		return 0
	}

	if node1 == nil || node2 == nil {
		return 1 // Cost of inserting/deleting a node
	}

	// If nodes are structurally identical, return 0
	if nodesStructurallyEqual(node1, node2) {
		return 0
	}

	// Calculate minimum cost between substitution, insertion, and deletion
	substitutionCost := 1 + calculateChildrenDistance(node1, node2)
	insertionCost := 1 + TreeEditDistance(nil, node2)
	deletionCost := 1 + TreeEditDistance(node1, nil)

	return mathutil.Min(substitutionCost, mathutil.Min(insertionCost, deletionCost))
}

// TokenSequenceSimilarity calculates similarity between two functions based on token sequences.
func TokenSequenceSimilarity(func1, func2 *ast.Function) float64 {
	if func1 == nil || func2 == nil {
		return 0.0
	}

	// Normalize the functions first
	norm1 := func1.Normalize()
	norm2 := func2.Normalize()

	if norm1 == nil || norm2 == nil {
		return 0.0
	}

	// Get normalized token sequences
	tokens1 := NormalizeTokenSequence(norm1)
	tokens2 := NormalizeTokenSequence(norm2)

	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}

	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	// Calculate Levenshtein distance between token sequences
	distance := LevenshteinDistance(strings.Join(tokens1, " "), strings.Join(tokens2, " "))
	maxLen := mathutil.Max(len(strings.Join(tokens1, " ")), len(strings.Join(tokens2, " ")))

	if maxLen == 0 {
		return 1.0
	}

	// Convert distance to similarity (0.0 to 1.0)
	return 1.0 - float64(distance)/float64(maxLen)
}

// LevenshteinDistance calculates the Levenshtein distance between two strings.
func LevenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}

	if len(s2) == 0 {
		return len(s1)
	}

	if s1 == s2 {
		return 0
	}

	// Create a matrix to store distances
	rows := len(s1) + 1
	cols := len(s2) + 1
	matrix := make([][]int, rows)

	for i := range rows {
		matrix[i] = make([]int, cols)
		matrix[i][0] = i
	}

	for j := range cols {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i < rows; i++ {
		for j := 1; j < cols; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = mathutil.Min(
				matrix[i-1][j]+1, // deletion
				mathutil.Min(
					matrix[i][j-1]+1,      // insertion
					matrix[i-1][j-1]+cost, // substitution
				),
			)
		}
	}

	return matrix[rows-1][cols-1]
}

// NormalizeTokenSequence extracts a normalized token sequence from a function.
func NormalizeTokenSequence(function *ast.Function) []string {
	if function == nil || function.AST == nil {
		return nil
	}

	// Generate source code from the AST
	fset := token.NewFileSet()
	var buf bytes.Buffer

	if err := format.Node(&buf, fset, function.AST); err != nil {
		return nil
	}

	source := buf.String()

	// Tokenize the source
	return tokenizeAndNormalize(source)
}

// tokenizeAndNormalize tokenizes source code and normalizes identifiers.
func tokenizeAndNormalize(source string) []string {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(source))

	var tokens []string
	s := scanner.Scanner{}
	s.Init(file, []byte(source), nil, scanner.ScanComments)

	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}

		// Skip whitespace and comments
		if tok == token.COMMENT {
			continue
		}

		// Normalize tokens
		switch tok {
		case token.IDENT:
			// Check if it's a basic type or keep as generic IDENT
			if isBasicType(lit) {
				tokens = append(tokens, lit)
			} else {
				tokens = append(tokens, "IDENT")
			}
		case token.INT, token.FLOAT:
			// Replace all numeric literals with generic NUMBER
			tokens = append(tokens, "NUMBER")
		case token.STRING:
			// Replace all string literals with generic STRING
			tokens = append(tokens, "STRING")
		case token.ILLEGAL, token.EOF, token.COMMENT, token.IMAG, token.CHAR,
			token.ADD, token.SUB, token.MUL, token.QUO, token.REM,
			token.AND, token.OR, token.XOR, token.SHL, token.SHR, token.AND_NOT,
			token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN, token.REM_ASSIGN,
			token.AND_ASSIGN, token.OR_ASSIGN, token.XOR_ASSIGN, token.SHL_ASSIGN, token.SHR_ASSIGN, token.AND_NOT_ASSIGN,
			token.LAND, token.LOR, token.ARROW, token.INC, token.DEC,
			token.EQL, token.LSS, token.GTR, token.ASSIGN, token.NOT, token.NEQ, token.LEQ, token.GEQ, token.DEFINE,
			token.ELLIPSIS, token.LPAREN, token.LBRACK, token.LBRACE, token.COMMA, token.PERIOD,
			token.RPAREN, token.RBRACK, token.RBRACE, token.SEMICOLON, token.COLON,
			token.BREAK, token.CASE, token.CHAN, token.CONST, token.CONTINUE, token.DEFAULT, token.DEFER,
			token.ELSE, token.FALLTHROUGH, token.FOR, token.FUNC, token.GO, token.GOTO, token.IF,
			token.IMPORT, token.INTERFACE, token.MAP, token.PACKAGE, token.RANGE, token.RETURN,
			token.SELECT, token.STRUCT, token.SWITCH, token.TYPE, token.VAR, token.TILDE:
			// Keep operators, keywords, and punctuation as-is
			if lit != "" {
				tokens = append(tokens, lit)
			} else {
				tokens = append(tokens, tok.String())
			}
		}

		_ = pos // unused
	}

	return tokens
}

// isBasicType checks if an identifier is a basic Go type.
func isBasicType(ident string) bool {
	basicTypes := map[string]bool{
		"int":        true,
		"int8":       true,
		"int16":      true,
		"int32":      true,
		"int64":      true,
		"uint":       true,
		"uint8":      true,
		"uint16":     true,
		"uint32":     true,
		"uint64":     true,
		"float32":    true,
		"float64":    true,
		"bool":       true,
		"string":     true,
		"byte":       true,
		"rune":       true,
		"complex64":  true,
		"complex128": true,
		"uintptr":    true,
	}
	return basicTypes[ident]
}

// nodesStructurallyEqual checks if two AST nodes are structurally equal.
//
//nolint:gocognit // Complex AST comparison algorithm acceptable
func nodesStructurallyEqual(node1, node2 goast.Node) bool {
	if node1 == nil && node2 == nil {
		return true
	}

	if node1 == nil || node2 == nil {
		return false
	}

	// Check if nodes have the same type
	type1 := getNodeType(node1)
	type2 := getNodeType(node2)

	if type1 != type2 {
		return false
	}

	// For simple nodes, check if they represent the same structure
	switch n1 := node1.(type) {
	case *goast.BinaryExpr:
		if n2, ok := node2.(*goast.BinaryExpr); ok {
			return n1.Op == n2.Op &&
				nodesStructurallyEqual(n1.X, n2.X) &&
				nodesStructurallyEqual(n1.Y, n2.Y)
		}
	case *goast.UnaryExpr:
		if n2, ok := node2.(*goast.UnaryExpr); ok {
			return n1.Op == n2.Op && nodesStructurallyEqual(n1.X, n2.X)
		}
	case *goast.CallExpr:
		if n2, ok := node2.(*goast.CallExpr); ok {
			if !nodesStructurallyEqual(n1.Fun, n2.Fun) {
				return false
			}
			if len(n1.Args) != len(n2.Args) {
				return false
			}
			for i, arg1 := range n1.Args {
				if !nodesStructurallyEqual(arg1, n2.Args[i]) {
					return false
				}
			}
			return true
		}
	case *goast.Ident:
		// For identifiers, we consider them equal if they are both identifiers
		// (ignoring the actual name for structural comparison)
		_, ok := node2.(*goast.Ident)
		return ok
	}

	// For other types, consider them equal if they have the same type
	return type1 == type2
}

// calculateChildrenDistance calculates the distance between children of two nodes.
func calculateChildrenDistance(node1, node2 goast.Node) int {
	children1 := getNodeChildren(node1)
	children2 := getNodeChildren(node2)

	if len(children1) == 0 && len(children2) == 0 {
		return 0
	}

	// Simple algorithm: calculate distance for each pair of children
	maxLen := mathutil.Max(len(children1), len(children2))
	totalDistance := 0

	for i := range maxLen {
		var child1, child2 goast.Node
		if i < len(children1) {
			child1 = children1[i]
		}
		if i < len(children2) {
			child2 = children2[i]
		}

		totalDistance += TreeEditDistance(child1, child2)
	}

	return totalDistance
}

// getNodeType returns a string representation of the node type.
func getNodeType(node goast.Node) string {
	switch node.(type) {
	case *goast.BinaryExpr:
		return "BinaryExpr"
	case *goast.UnaryExpr:
		return "UnaryExpr"
	case *goast.CallExpr:
		return "CallExpr"
	case *goast.Ident:
		return "Ident"
	case *goast.BasicLit:
		return "BasicLit"
	case *goast.ReturnStmt:
		return "ReturnStmt"
	case *goast.AssignStmt:
		return "AssignStmt"
	case *goast.ExprStmt:
		return "ExprStmt"
	case *goast.IfStmt:
		return "IfStmt"
	case *goast.ForStmt:
		return "ForStmt"
	case *goast.BlockStmt:
		return "BlockStmt"
	default:
		return "Unknown"
	}
}

// getNodeChildren returns the child nodes of a given AST node.
//
//nolint:gocognit // Complex AST traversal algorithm acceptable
func getNodeChildren(node goast.Node) []goast.Node {
	var children []goast.Node

	switch n := node.(type) {
	case *goast.BinaryExpr:
		children = append(children, n.X, n.Y)
	case *goast.UnaryExpr:
		children = append(children, n.X)
	case *goast.CallExpr:
		children = append(children, n.Fun)
		for _, arg := range n.Args {
			children = append(children, arg)
		}
	case *goast.ReturnStmt:
		for _, result := range n.Results {
			children = append(children, result)
		}
	case *goast.AssignStmt:
		for _, lhs := range n.Lhs {
			children = append(children, lhs)
		}
		for _, rhs := range n.Rhs {
			children = append(children, rhs)
		}
	case *goast.ExprStmt:
		children = append(children, n.X)
	case *goast.BlockStmt:
		for _, stmt := range n.List {
			children = append(children, stmt)
		}
	case *goast.IfStmt:
		if n.Init != nil {
			children = append(children, n.Init)
		}
		children = append(children, n.Cond)
		children = append(children, n.Body)
		if n.Else != nil {
			children = append(children, n.Else)
		}
	case *goast.ForStmt:
		if n.Init != nil {
			children = append(children, n.Init)
		}
		if n.Cond != nil {
			children = append(children, n.Cond)
		}
		if n.Post != nil {
			children = append(children, n.Post)
		}
		children = append(children, n.Body)
	}

	return children
}

// Helper functions have been moved to pkg/mathutil package.
