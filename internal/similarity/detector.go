package similarity

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	goast "go/ast"
	"go/format"
	"go/token"

	"github.com/paveg/similarity-go/internal/ast"
	"github.com/paveg/similarity-go/internal/config"
	"github.com/paveg/similarity-go/pkg/mathutil"
)

// Detector handles similarity detection between functions.
type Detector struct {
	threshold       float64
	config          *config.Config
	similarityCache map[string]float64 // Cache for similarity results
}

// Match represents a match between two similar functions.
type Match struct {
	Function1  *ast.Function
	Function2  *ast.Function
	Similarity float64
}

// NewDetector creates a new similarity detector with the given threshold and configuration.
func NewDetector(threshold float64) *Detector {
	cfg, _ := config.Load("") // Load default config, ignore errors for backward compatibility
	return &Detector{
		threshold:       threshold,
		config:          cfg,
		similarityCache: make(map[string]float64),
	}
}

// NewDetectorWithConfig creates a new similarity detector with explicit configuration.
func NewDetectorWithConfig(threshold float64, cfg *config.Config) *Detector {
	return &Detector{
		threshold:       threshold,
		config:          cfg,
		similarityCache: make(map[string]float64),
	}
}

// CalculateSimilarity calculates the similarity between two functions
// Returns a value between 0.0 (completely different) and 1.0 (identical).
func (d *Detector) CalculateSimilarity(func1, func2 *ast.Function) float64 {
	if func1 == nil || func2 == nil {
		return 0.0
	}

	// Early termination: check structural hashes first for quick identical/different detection
	hash1 := func1.Hash()
	hash2 := func2.Hash()
	if hash1 == hash2 {
		return 1.0 // Identical structural hash means identical functions
	}

	// Check cache for previously calculated similarity
	cacheKey := d.getCacheKey(hash1, hash2)
	if cached, exists := d.similarityCache[cacheKey]; exists {
		return cached
	}

	// Early termination: quick signature-based filtering
	if !d.couldBeSimilar(func1, func2) {
		// Cache the result (with size limit)
		if len(d.similarityCache) < d.config.Similarity.Limits.MaxCacheSize {
			d.similarityCache[cacheKey] = 0.0
		}
		return 0.0 // Functions are too different to be similar
	}

	// If both functions have the same normalized AST structure, they are identical
	if d.compareNormalizedAST(func1, func2) {
		return 1.0
	}

	// Use multiple similarity metrics and combine them
	// 1. Tree edit distance similarity
	treeEditSim := d.calculateTreeEditSimilarity(func1, func2)

	// 2. Token sequence similarity
	tokenSim := TokenSequenceSimilarity(func1, func2)

	// 3. Structural similarity (existing algorithm)
	structuralSim := d.calculateStructuralSimilarity(func1, func2)

	// 4. Signature similarity
	signatureSim := d.calculateSignatureSimilarity(func1, func2)

	// Weighted combination: prioritize tree edit and token similarity
	// as they are more sophisticated algorithms
	weights := d.config.Similarity.Weights
	result := weights.TreeEdit*treeEditSim + weights.TokenSimilarity*tokenSim + weights.Structural*structuralSim + weights.Signature*signatureSim

	// Cache the result for future use (with size limit)
	if len(d.similarityCache) < d.config.Similarity.Limits.MaxCacheSize {
		d.similarityCache[cacheKey] = result
	}

	return result
}

// IsAboveThreshold checks if similarity is above the configured threshold.
func (d *Detector) IsAboveThreshold(similarity float64) bool {
	return similarity >= d.threshold
}

// FindSimilarFunctions finds all similar function pairs above the threshold.
func (d *Detector) FindSimilarFunctions(functions []*ast.Function) []Match {
	var matches []Match

	for i := range functions {
		for j := i + 1; j < len(functions); j++ {
			similarity := d.CalculateSimilarity(functions[i], functions[j])
			if d.IsAboveThreshold(similarity) {
				matches = append(matches, Match{
					Function1:  functions[i],
					Function2:  functions[j],
					Similarity: similarity,
				})
			}
		}
	}

	return matches
}

// couldBeSimilar performs quick heuristic checks to filter out obviously dissimilar functions.
// This avoids expensive similarity calculations for functions that are clearly different.
func (d *Detector) couldBeSimilar(func1, func2 *ast.Function) bool {
	// Check signature length difference
	sig1 := func1.GetSignature()
	sig2 := func2.GetSignature()

	if mathutil.Abs(len(sig1)-len(sig2)) > d.config.Similarity.Limits.MaxSignatureLengthDiff {
		return false
	}

	// Check line count ratio
	lines1 := func1.LineCount
	lines2 := func2.LineCount

	if lines1 == 0 || lines2 == 0 {
		return true // Can't determine, let full comparison decide
	}

	ratio := float64(lines1) / float64(lines2)
	maxRatio := d.config.Similarity.Limits.MaxLineDifferenceRatio
	if ratio > maxRatio || ratio < 1.0/maxRatio {
		return false
	}

	// Check basic structural compatibility
	if func1.AST != nil && func2.AST != nil {
		// Both have bodies, check statement count difference
		if func1.AST.Body != nil && func2.AST.Body != nil {
			stmt1Count := len(func1.AST.Body.List)
			stmt2Count := len(func2.AST.Body.List)

			// If one is empty and other has many statements, likely different
			maxEmpty := d.config.Processing.MaxEmptyVsPopulated
			if (stmt1Count == 0 && stmt2Count > maxEmpty) || (stmt2Count == 0 && stmt1Count > maxEmpty) {
				return false
			}
		}
	}

	return true // Passed quick checks, allow full comparison
}

// getCacheKey creates a consistent cache key for two function hashes.
// Always puts the smaller hash first to ensure (A,B) and (B,A) have the same key.
func (d *Detector) getCacheKey(hash1, hash2 string) string {
	return mathutil.CreateConsistentKey(hash1, hash2)
}

// compareNormalizedAST compares the normalized AST structures of two functions.
func (d *Detector) compareNormalizedAST(func1, func2 *ast.Function) bool {
	norm1 := func1.Normalize()
	norm2 := func2.Normalize()

	if norm1 == nil || norm2 == nil {
		return false
	}

	if norm1.AST == nil || norm2.AST == nil {
		return false
	}

	// Generate hashes of the normalized ASTs
	hash1 := d.generateASTHash(norm1.AST)
	hash2 := d.generateASTHash(norm2.AST)

	return hash1 == hash2
}

// calculateTreeEditSimilarity calculates similarity based on tree edit distance.
func (d *Detector) calculateTreeEditSimilarity(func1, func2 *ast.Function) float64 {
	if func1 == nil || func2 == nil || func1.AST == nil || func2.AST == nil {
		return 0.0
	}

	// Use normalized functions for comparison
	norm1 := func1.Normalize()
	norm2 := func2.Normalize()

	if norm1 == nil || norm2 == nil || norm1.AST == nil || norm2.AST == nil {
		return 0.0
	}

	// Calculate tree edit distance
	distance := TreeEditDistance(norm1.AST, norm2.AST)

	// Convert distance to similarity
	// Estimate maximum possible distance as sum of node counts
	maxNodes := d.countASTNodes(norm1.AST) + d.countASTNodes(norm2.AST)
	if maxNodes == 0 {
		return 1.0
	}

	// Convert to similarity score (0.0 to 1.0)
	similarity := 1.0 - float64(distance)/float64(maxNodes)
	if similarity < 0.0 {
		similarity = 0.0
	}

	return similarity
}

// countASTNodes counts the number of nodes in an AST.
//
//nolint:gocognit // Complex AST node counting algorithm acceptable
func (d *Detector) countASTNodes(node goast.Node) int {
	if node == nil {
		return 0
	}

	count := 1 // Count the current node

	// Add counts for child nodes
	switch n := node.(type) {
	case *goast.FuncDecl:
		if n.Type != nil {
			count += d.countASTNodes(n.Type)
		}
		if n.Body != nil {
			count += d.countASTNodes(n.Body)
		}
	case *goast.BlockStmt:
		for _, stmt := range n.List {
			count += d.countASTNodes(stmt)
		}
	case *goast.BinaryExpr:
		count += d.countASTNodes(n.X)
		count += d.countASTNodes(n.Y)
	case *goast.UnaryExpr:
		count += d.countASTNodes(n.X)
	case *goast.CallExpr:
		count += d.countASTNodes(n.Fun)
		for _, arg := range n.Args {
			count += d.countASTNodes(arg)
		}
	case *goast.ReturnStmt:
		for _, result := range n.Results {
			count += d.countASTNodes(result)
		}
	case *goast.AssignStmt:
		for _, lhs := range n.Lhs {
			count += d.countASTNodes(lhs)
		}
		for _, rhs := range n.Rhs {
			count += d.countASTNodes(rhs)
		}
	case *goast.ExprStmt:
		count += d.countASTNodes(n.X)
	case *goast.IfStmt:
		if n.Init != nil {
			count += d.countASTNodes(n.Init)
		}
		count += d.countASTNodes(n.Cond)
		count += d.countASTNodes(n.Body)
		if n.Else != nil {
			count += d.countASTNodes(n.Else)
		}
	case *goast.ForStmt:
		if n.Init != nil {
			count += d.countASTNodes(n.Init)
		}
		if n.Cond != nil {
			count += d.countASTNodes(n.Cond)
		}
		if n.Post != nil {
			count += d.countASTNodes(n.Post)
		}
		count += d.countASTNodes(n.Body)
	}

	return count
}

// calculateStructuralSimilarity compares the structural elements of functions.
func (d *Detector) calculateStructuralSimilarity(func1, func2 *ast.Function) float64 {
	if func1.AST == nil || func2.AST == nil {
		return 0.0
	}

	// Compare function signatures (excluding parameter names)
	sig1 := d.getStructuralSignature(func1)
	sig2 := d.getStructuralSignature(func2)

	if sig1 == sig2 {
		// Same signature, compare body structure
		return d.compareBodyStructure(func1.AST.Body, func2.AST.Body)
	}

	// Check if they are similar operations (add vs multiply, etc.)
	bodyScore := d.compareBodyStructure(func1.AST.Body, func2.AST.Body)

	// If functions have different signatures but similar body structure,
	// return lower similarity based on the operation similarity
	if bodyScore > 0.7 && d.hasSimilarOperations(func1, func2) {
		return d.config.Similarity.Thresholds.DefaultSimilarOperations
	}

	return bodyScore * d.config.Similarity.Weights.DifferentSignature // Different signatures result in lower similarity
}

// calculateSignatureSimilarity compares function signatures.
func (d *Detector) calculateSignatureSimilarity(func1, func2 *ast.Function) float64 {
	sig1 := func1.GetSignature()
	sig2 := func2.GetSignature()

	if sig1 == sig2 {
		return 1.0
	}

	// Simple string similarity based on signature length and common characters
	return d.stringSimilarity(sig1, sig2)
}

// getStructuralSignature gets signature without parameter names.
//
//nolint:gocognit // Complex structural signature generation algorithm acceptable
func (d *Detector) getStructuralSignature(fn *ast.Function) string {
	if fn.AST == nil || fn.AST.Type == nil {
		return ""
	}

	// Create a simplified signature focusing on types, not names
	var paramTypes []string

	if fn.AST.Type.Params != nil {
		for _, param := range fn.AST.Type.Params.List {
			if param.Type != nil {
				paramTypes = append(paramTypes, d.typeToString(param.Type))
			}
		}
	}

	resultTypes := ""
	resultCount := 0

	if fn.AST.Type.Results != nil {
		for _, result := range fn.AST.Type.Results.List {
			if result.Type != nil {
				if resultTypes != "" {
					resultTypes += ", "
				}

				resultTypes += d.typeToString(result.Type)
				resultCount++
			}
		}
	}

	signature := "func("

	for i, pt := range paramTypes {
		if i > 0 {
			signature += ", "
		}

		signature += pt
	}

	signature += ")"

	if resultTypes != "" {
		if resultCount > 1 {
			signature += " (" + resultTypes + ")"
		} else {
			signature += " " + resultTypes
		}
	}

	return signature
}

// compareBodyStructure compares the structure of function bodies.
func (d *Detector) compareBodyStructure(body1, body2 *goast.BlockStmt) float64 {
	if body1 == nil && body2 == nil {
		return 1.0
	}

	if body1 == nil || body2 == nil {
		return 0.0
	}

	// Simple structural comparison based on statement counts and types
	if len(body1.List) != len(body2.List) {
		return d.config.Similarity.Thresholds.StatementCountPenalty // Different number of statements
	}

	matches := 0

	for i := 0; i < len(body1.List) && i < len(body2.List); i++ {
		if d.statementsStructurallyEqual(body1.List[i], body2.List[i]) {
			matches++
		}
	}

	if len(body1.List) == 0 {
		return 1.0
	}

	return float64(matches) / float64(len(body1.List))
}

// statementsStructurallyEqual checks if two statements have the same structure.
func (d *Detector) statementsStructurallyEqual(stmt1, stmt2 goast.Stmt) bool {
	// Simple type-based comparison
	return fmt.Sprintf("%T", stmt1) == fmt.Sprintf("%T", stmt2)
}

// generateASTHash generates a hash from an AST node.
func (d *Detector) generateASTHash(node goast.Node) string {
	if node == nil {
		return ""
	}

	// Generate source code from AST and hash it
	fset := token.NewFileSet()

	var buf bytes.Buffer

	if err := format.Node(&buf, fset, node); err != nil {
		return "hash_error"
	}

	hash := sha256.Sum256(buf.Bytes())

	return hex.EncodeToString(hash[:])
}

// typeToString converts an AST type to its string representation.
func (d *Detector) typeToString(expr goast.Expr) string {
	switch t := expr.(type) {
	case *goast.Ident:
		return t.Name
	case *goast.StarExpr:
		return "*" + d.typeToString(t.X)
	case *goast.SelectorExpr:
		return d.typeToString(t.X) + "." + t.Sel.Name
	default:
		return "unknown"
	}
}

// stringSimilarity calculates simple string similarity.
func (d *Detector) stringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	if len(s1) == 0 && len(s2) == 0 {
		return 1.0
	}

	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Simple similarity based on length difference
	maxLen := max(len(s1), len(s2))

	diff := mathutil.Abs(len(s1) - len(s2))

	return 1.0 - float64(diff)/float64(maxLen)
}

// hasSimilarOperations checks if functions perform similar operations.
func (d *Detector) hasSimilarOperations(func1, func2 *ast.Function) bool {
	// Check if both functions have similar operation patterns (e.g., binary expressions)
	// This provides a basic heuristic for operational similarity
	return d.hasBinaryExpressions(func1.AST.Body) && d.hasBinaryExpressions(func2.AST.Body)
}

// hasBinaryExpressions checks if a block contains binary expressions.
func (d *Detector) hasBinaryExpressions(body *goast.BlockStmt) bool {
	if body == nil {
		return false
	}

	for _, stmt := range body.List {
		if retStmt, ok := stmt.(*goast.ReturnStmt); ok {
			for _, result := range retStmt.Results {
				if _, isBinaryExpr := result.(*goast.BinaryExpr); isBinaryExpr {
					return true
				}
			}
		}
	}

	return false
}

// abs function has been moved to pkg/mathutil package.
