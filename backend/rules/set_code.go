package rules

import (
	"strings"

	"github.com/expr-lang/expr/ast"
)

// setCodePatcher makes comparisons against the `set` field case-insensitive.
// The evaluator exposes `set` in uppercase, so this uppercases string literals
// compared against it: `set == "c20"`, `"c20" == set`, `set in ["c20", "mh3"]`.
type setCodePatcher struct{}

func (setCodePatcher) Visit(node *ast.Node) {
	binary, ok := (*node).(*ast.BinaryNode)
	if !ok {
		return
	}
	if isSetIdentifier(binary.Left) {
		uppercaseStringLiterals(binary.Right)
	} else if isSetIdentifier(binary.Right) {
		uppercaseStringLiterals(binary.Left)
	}
}

func isSetIdentifier(node ast.Node) bool {
	identifier, ok := node.(*ast.IdentifierNode)
	return ok && identifier.Value == "set"
}

func uppercaseStringLiterals(node ast.Node) {
	switch n := node.(type) {
	case *ast.StringNode:
		n.Value = strings.ToUpper(n.Value)
	case *ast.ArrayNode:
		for _, element := range n.Nodes {
			uppercaseStringLiterals(element)
		}
	}
}
