package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root := "."

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || shouldIgnoreFile(path) {
			return nil
		}

		fileSet := token.NewFileSet()
		node, err := parser.ParseFile(fileSet, path, nil, parser.AllErrors)
		if err != nil {
			return nil
		}

		ast.Inspect(node, func(n ast.Node) bool {
			if ret, ok := n.(*ast.ReturnStmt); ok {
				checkForUnwrappedErrors(ret, fileSet, path, node)
			}
			return true
		})

		return nil
	})

	if err != nil {
		fmt.Println("Error walking files:", err)
		os.Exit(1)
	}
}

func shouldIgnoreFile(path string) bool {
	return strings.HasPrefix(path, "vendor/") ||
		strings.HasSuffix(path, ".gen.go") ||
		!strings.HasSuffix(path, ".go")
}

func checkForUnwrappedErrors(ret *ast.ReturnStmt, fset *token.FileSet, path string, rootNode ast.Node) {
	for _, result := range ret.Results {
		if id, ok := result.(*ast.Ident); ok && id.Name == "err" {
			line := fset.Position(id.Pos()).Line
			context := inferContextFromFunction(ret, rootNode)
			suggestion := fmt.Sprintf("%s:%d:fmt.Errorf(\"%s: %%w\", err)", path, line, context)
			fmt.Println(suggestion)
		}
	}
}

func inferContextFromFunction(n ast.Node, rootNode ast.Node) string {
	// Walk the AST to find the enclosing function declaration
	var functionName string
	ast.Inspect(rootNode, func(node ast.Node) bool {
		if fn, ok := node.(*ast.FuncDecl); ok {
			// Check if our return statement is within this function
			if n.Pos() >= fn.Pos() && n.End() <= fn.End() {
				functionName = fn.Name.Name
				return false // Stop inspecting further once the function is found
			}
		}
		return true
	})

	if functionName == "" {
		return "operation" // Default if no function found
	}

	return formatFunctionName(functionName)
}

func formatFunctionName(name string) string {
	var result []rune
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, ' ')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
