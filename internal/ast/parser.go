package ast

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"

	"github.com/paveg/similarity-go/pkg/types"
)

// Parser handles parsing Go source files and extracting function information.
type Parser struct {
	fileSet *token.FileSet
}

// ParseResult contains the results of parsing one or more Go files.
type ParseResult struct {
	Functions []*Function  // Successfully parsed functions
	Errors    []error      // Errors encountered during parsing
	Metadata  FileMetadata // Additional metadata about the parsing operation
}

// FileMetadata contains metadata about the parsing operation.
type FileMetadata struct {
	TotalFiles      int // Total number of files processed
	SuccessfulFiles int // Number of files parsed successfully
	FailedFiles     int // Number of files that failed to parse
}

// NewParser creates a new Parser instance.
func NewParser() *Parser {
	return &Parser{
		fileSet: token.NewFileSet(),
	}
}

// ParseFile parses a single Go source file and extracts function information.
func (p *Parser) ParseFile(filename string) types.Result[*ParseResult] {
	// Read the file
	src, err := os.ReadFile(filename)
	if err != nil {
		return types.Err[*ParseResult](err)
	}

	// Parse the file
	file, err := parser.ParseFile(p.fileSet, filename, src, parser.ParseComments)
	if err != nil {
		return types.Err[*ParseResult](err)
	}

	// Extract functions
	functions := p.extractFunctions(file, filename)

	return types.Ok(&ParseResult{
		Functions: functions,
		Errors:    []error{},
		Metadata: FileMetadata{
			TotalFiles:      1,
			SuccessfulFiles: 1,
			FailedFiles:     0,
		},
	})
}

// ParseFiles parses multiple Go source files and returns combined results.
// This method continues processing even if some files fail, collecting all errors.
func (p *Parser) ParseFiles(filenames []string) types.Result[*ParseResult] {
	var allFunctions []*Function
	var allErrors []error
	successCount := 0

	for _, filename := range filenames {
		result := p.ParseFile(filename)
		if result.IsOk() {
			parseResult := result.Unwrap()
			allFunctions = append(allFunctions, parseResult.Functions...)
			successCount++
		} else {
			allErrors = append(allErrors, result.Error())
		}
	}

	return types.Ok(&ParseResult{
		Functions: allFunctions,
		Errors:    allErrors,
		Metadata: FileMetadata{
			TotalFiles:      len(filenames),
			SuccessfulFiles: successCount,
			FailedFiles:     len(filenames) - successCount,
		},
	})
}

// extractFunctions extracts all function declarations from an AST file.
func (p *Parser) extractFunctions(file *ast.File, filename string) []*Function {
	var functions []*Function

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			// Skip interface method declarations (they have no body)
			if node.Body == nil {
				return true
			}

			fn := p.createFunction(node, filename)
			functions = append(functions, fn)
		}
		return true
	})

	return functions
}

// createFunction creates a Function instance from an AST function declaration.
func (p *Parser) createFunction(funcDecl *ast.FuncDecl, filename string) *Function {
	startPos := p.fileSet.Position(funcDecl.Pos())
	endPos := p.fileSet.Position(funcDecl.End())

	lineCount := endPos.Line - startPos.Line + 1

	return &Function{
		Name:      funcDecl.Name.Name,
		File:      filename,
		StartLine: startPos.Line,
		EndLine:   endPos.Line,
		AST:       funcDecl,
		LineCount: lineCount,
	}
}
