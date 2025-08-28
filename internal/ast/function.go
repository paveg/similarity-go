package ast

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
)

// Function represents a Go function with its metadata and AST representation.
type Function struct {
	Name       string        // Function name
	File       string        // Source file path
	StartLine  int           // Starting line number
	EndLine    int           // Ending line number
	AST        *ast.FuncDecl // Original AST node
	Normalized *ast.FuncDecl // Normalized AST for comparison
	hash       string        // Cached structure hash
	signature  string        // Cached function signature
	LineCount  int           // Number of lines in the function
}

// GetSignature returns the function signature as a string.
// The signature is cached after first computation.
func (f *Function) GetSignature() string {
	if f.signature != "" {
		return f.signature
	}

	if f.AST == nil || f.AST.Type == nil {
		f.signature = "func()"

		return f.signature
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), f.AST.Type); err != nil {
		f.signature = "func()"

		return f.signature
	}

	f.signature = buf.String()

	return f.signature
}

// GetSource returns the complete source code of the function.
func (f *Function) GetSource() (string, error) {
	if f.AST == nil {
		return "", nil
	}

	var buf bytes.Buffer

	err := format.Node(&buf, token.NewFileSet(), f.AST)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// IsValid checks if the function meets the minimum requirements for analysis.
// It must have a non-nil body and meet the minimum line count.
func (f *Function) IsValid(minLines int) bool {
	// Check if function has a body (not an interface method declaration)
	if f.AST == nil || f.AST.Body == nil {
		return false
	}

	// Check minimum line count
	if f.LineCount < minLines {
		return false
	}

	return true
}

// Hash returns a structural hash of the function for quick comparison.
func (f *Function) Hash() string {
	if f.hash != "" {
		return f.hash
	}

	if f.AST == nil {
		// Include function name in hash even for nil AST
		hashComponents := []string{
			f.Name,
			f.GetSignature(),
			fmt.Sprintf("lines:%d-%d", f.StartLine, f.EndLine),
			fmt.Sprintf("count:%d", f.LineCount),
			"nil_ast",
		}

		combined := fmt.Sprintf("%v", hashComponents)
		hasher := sha256.New()
		hasher.Write([]byte(combined))
		f.hash = hex.EncodeToString(hasher.Sum(nil))[:16]

		return f.hash
	}

	// Create a structural hash based on function signature and basic structure
	hashComponents := []string{
		f.Name,
		f.GetSignature(),
		fmt.Sprintf("lines:%d-%d", f.StartLine, f.EndLine),
		fmt.Sprintf("count:%d", f.LineCount),
	}

	// Add body structure if available
	if f.AST.Body != nil {
		hashComponents = append(hashComponents, fmt.Sprintf("stmts:%d", len(f.AST.Body.List)))
	}

	// Combine all components and create SHA256 hash
	combined := fmt.Sprintf("%v", hashComponents)
	hasher := sha256.New()
	hasher.Write([]byte(combined))
	f.hash = hex.EncodeToString(hasher.Sum(nil))[:16] // Use first 16 chars for readability

	return f.hash
}

// Normalize returns a normalized version of the function for comparison.
// Normalization removes variable names, literal values, and other non-structural elements
// while preserving the essential structure for similarity comparison.
func (f *Function) Normalize() *Function {
	if f.Normalized != nil {
		return &Function{
			Name:       f.Name,
			File:       f.File,
			StartLine:  f.StartLine,
			EndLine:    f.EndLine,
			AST:        f.Normalized,
			Normalized: f.Normalized,
			LineCount:  f.LineCount,
		}
	}

	if f.AST == nil {
		return f
	}

	// Create a deep copy of the AST and normalize it
	normalizedAST := f.deepCopyFuncDecl(f.AST)
	f.normalizeNode(normalizedAST)
	f.Normalized = normalizedAST

	return &Function{
		Name:       f.Name,
		File:       f.File,
		StartLine:  f.StartLine,
		EndLine:    f.EndLine,
		AST:        f.Normalized,
		Normalized: f.Normalized,
		LineCount:  f.LineCount,
	}
}

// deepCopyFuncDecl creates a deep copy of a FuncDecl.
func (f *Function) deepCopyFuncDecl(original *ast.FuncDecl) *ast.FuncDecl {
	if original == nil {
		return nil
	}

	// Create new function declaration
	copied := &ast.FuncDecl{
		Doc:  original.Doc,
		Recv: original.Recv,
		Name: original.Name,
		Type: original.Type,
		Body: f.deepCopyBlockStmt(original.Body),
	}

	return copied
}

// deepCopyBlockStmt creates a deep copy of a BlockStmt.
func (f *Function) deepCopyBlockStmt(original *ast.BlockStmt) *ast.BlockStmt {
	if original == nil {
		return nil
	}

	copied := &ast.BlockStmt{
		Lbrace: original.Lbrace,
		Rbrace: original.Rbrace,
		List:   make([]ast.Stmt, len(original.List)),
	}

	for i, stmt := range original.List {
		copied.List[i] = f.deepCopyStmt(stmt)
	}

	return copied
}

// deepCopyStmt creates a deep copy of a statement.
func (f *Function) deepCopyStmt(original ast.Stmt) ast.Stmt {
	//nolint:errcheck // Type assertion is safe: we know the input is ast.Stmt and return ast.Stmt
	return f.deepCopyNode(original, func(node ast.Node) ast.Node {
		switch stmt := node.(type) {
		case *ast.ReturnStmt:
			return f.copyReturnStmt(stmt)
		case *ast.AssignStmt:
			return f.copyAssignStmt(stmt)
		case *ast.ExprStmt:
			return &ast.ExprStmt{X: f.deepCopyExpr(stmt.X)}
		case *ast.IfStmt:
			return f.copyIfStmt(stmt)
		case *ast.ForStmt:
			return f.copyForStmt(stmt)
		case *ast.BlockStmt:
			return f.deepCopyBlockStmt(stmt)
		default:
			return stmt
		}
	}).(ast.Stmt)
}

// copyReturnStmt creates a deep copy of a return statement.
func (f *Function) copyReturnStmt(stmt *ast.ReturnStmt) *ast.ReturnStmt {
	copied := &ast.ReturnStmt{
		Return:  stmt.Return,
		Results: make([]ast.Expr, len(stmt.Results)),
	}
	for i, result := range stmt.Results {
		copied.Results[i] = f.deepCopyExpr(result)
	}
	return copied
}

// copyAssignStmt creates a deep copy of an assignment statement.
func (f *Function) copyAssignStmt(stmt *ast.AssignStmt) *ast.AssignStmt {
	copied := &ast.AssignStmt{
		Lhs:    make([]ast.Expr, len(stmt.Lhs)),
		TokPos: stmt.TokPos,
		Tok:    stmt.Tok,
		Rhs:    make([]ast.Expr, len(stmt.Rhs)),
	}
	for i, lhs := range stmt.Lhs {
		copied.Lhs[i] = f.deepCopyExpr(lhs)
	}
	for i, rhs := range stmt.Rhs {
		copied.Rhs[i] = f.deepCopyExpr(rhs)
	}
	return copied
}

// copyIfStmt creates a deep copy of an if statement.
func (f *Function) copyIfStmt(stmt *ast.IfStmt) *ast.IfStmt {
	var initStmt ast.Stmt
	if stmt.Init != nil {
		initStmt = f.deepCopyStmt(stmt.Init)
	}

	var elseStmt ast.Stmt
	if stmt.Else != nil {
		elseStmt = f.deepCopyStmt(stmt.Else)
	}

	return &ast.IfStmt{
		If:   stmt.If,
		Init: initStmt,
		Cond: f.deepCopyExpr(stmt.Cond),
		Body: f.deepCopyBlockStmt(stmt.Body),
		Else: elseStmt,
	}
}

// copyForStmt creates a deep copy of a for statement.
func (f *Function) copyForStmt(stmt *ast.ForStmt) *ast.ForStmt {
	var initStmt ast.Stmt
	if stmt.Init != nil {
		initStmt = f.deepCopyStmt(stmt.Init)
	}

	var condExpr ast.Expr
	if stmt.Cond != nil {
		condExpr = f.deepCopyExpr(stmt.Cond)
	}

	var postStmt ast.Stmt
	if stmt.Post != nil {
		postStmt = f.deepCopyStmt(stmt.Post)
	}

	return &ast.ForStmt{
		For:  stmt.For,
		Init: initStmt,
		Cond: condExpr,
		Post: postStmt,
		Body: f.deepCopyBlockStmt(stmt.Body),
	}
}

// deepCopyExpr creates a deep copy of an expression.
func (f *Function) deepCopyExpr(original ast.Expr) ast.Expr {
	//nolint:errcheck // Type assertion is safe: we know the input is ast.Expr and return ast.Expr
	return f.deepCopyNode(original, func(node ast.Node) ast.Node {
		switch expr := node.(type) {
		case *ast.Ident:
			return f.copyIdent(expr)
		case *ast.BasicLit:
			return f.copyBasicLit(expr)
		case *ast.BinaryExpr:
			return f.copyBinaryExpr(expr)
		case *ast.UnaryExpr:
			return f.copyUnaryExpr(expr)
		case *ast.CallExpr:
			return f.copyCallExpr(expr)
		default:
			return expr
		}
	}).(ast.Expr)
}

// deepCopyNode provides common nil-check and type conversion pattern for copying AST nodes.
func (f *Function) deepCopyNode(original ast.Node, copyFunc func(ast.Node) ast.Node) ast.Node {
	if original == nil {
		return nil
	}
	return copyFunc(original)
}

// copyIdent creates a deep copy of an identifier.
func (f *Function) copyIdent(expr *ast.Ident) *ast.Ident {
	return &ast.Ident{
		NamePos: expr.NamePos,
		Name:    expr.Name,
		Obj:     expr.Obj,
	}
}

// copyBasicLit creates a deep copy of a basic literal.
func (f *Function) copyBasicLit(expr *ast.BasicLit) *ast.BasicLit {
	return &ast.BasicLit{
		ValuePos: expr.ValuePos,
		Kind:     expr.Kind,
		Value:    expr.Value,
	}
}

// copyBinaryExpr creates a deep copy of a binary expression.
func (f *Function) copyBinaryExpr(expr *ast.BinaryExpr) *ast.BinaryExpr {
	return &ast.BinaryExpr{
		X:     f.deepCopyExpr(expr.X),
		OpPos: expr.OpPos,
		Op:    expr.Op,
		Y:     f.deepCopyExpr(expr.Y),
	}
}

// copyUnaryExpr creates a deep copy of a unary expression.
func (f *Function) copyUnaryExpr(expr *ast.UnaryExpr) *ast.UnaryExpr {
	return &ast.UnaryExpr{
		OpPos: expr.OpPos,
		Op:    expr.Op,
		X:     f.deepCopyExpr(expr.X),
	}
}

// copyCallExpr creates a deep copy of a call expression.
func (f *Function) copyCallExpr(expr *ast.CallExpr) *ast.CallExpr {
	copied := &ast.CallExpr{
		Fun:      f.deepCopyExpr(expr.Fun),
		Lparen:   expr.Lparen,
		Args:     make([]ast.Expr, len(expr.Args)),
		Ellipsis: expr.Ellipsis,
		Rparen:   expr.Rparen,
	}
	for i, arg := range expr.Args {
		copied.Args[i] = f.deepCopyExpr(arg)
	}
	return copied
}

// normalizeNode recursively normalizes AST nodes by replacing variable names
// and literal values with standardized placeholders.
func (f *Function) normalizeNode(node ast.Node) {
	if node == nil {
		return
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.Ident:
			// Replace variable names with normalized placeholders
			if node.Obj != nil {
				switch node.Obj.Kind {
				case ast.Var:
					node.Name = "VAR"
				case ast.Con:
					node.Name = "CONST"
				case ast.Fun:
					node.Name = "FUNC"
				case ast.Typ:
					node.Name = "TYPE"
				case ast.Bad, ast.Pkg, ast.Lbl:
					// Leave these types unchanged
				}
			} else if !f.isBuiltinIdentifier(node.Name) {
				// Only normalize non-builtin identifiers
				node.Name = "ID"
			}

		case *ast.BasicLit:
			// Normalize literal values to their types
			// Only normalize basic literal tokens, leave others unchanged
			//nolint:exhaustive // We only want to handle literal types, not all token types
			switch node.Kind {
			case token.INT:
				node.Value = "0"
			case token.FLOAT:
				node.Value = "0.0"
			case token.STRING:
				node.Value = `""`
			case token.CHAR:
				node.Value = `'_'`
			case token.IMAG:
				node.Value = "0i"
			}
		}
		return true
	})
}

// isBuiltinIdentifier checks if an identifier is a Go builtin.
func (f *Function) isBuiltinIdentifier(name string) bool {
	builtins := map[string]bool{
		// Go builtin types
		"bool": true, "byte": true, "complex64": true, "complex128": true,
		"error": true, "float32": true, "float64": true, "int": true,
		"int8": true, "int16": true, "int32": true, "int64": true,
		"rune": true, "string": true, "uint": true, "uint8": true,
		"uint16": true, "uint32": true, "uint64": true, "uintptr": true,

		// Go builtin functions
		"append": true, "cap": true, "close": true, "complex": true,
		"copy": true, "delete": true, "imag": true, "len": true,
		"make": true, "new": true, "panic": true, "print": true,
		"println": true, "real": true, "recover": true,

		// Go constants
		"true": true, "false": true, "iota": true, "nil": true,
	}

	return builtins[name]
}
