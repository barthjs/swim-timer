package i18n

import (
	"reflect"
	"text/template/parse"
)

// missingFunc names the template function BlankMissing appends to every output action.
const missingFunc = "emptyIfMissing"

// TemplateFuncs returns the functions a template needs before BlankMissing can rewrite it.
// Register them with Funcs before parsing.
func TemplateFuncs() map[string]any {
	return map[string]any{missingFunc: emptyIfMissing}
}

// BlankMissing makes a parsed text or HTML template print nothing instead of "<no value>" for a missing value
// and runs on every template of a set between parsing and the first execution.
func BlankMissing(tree *parse.Tree) {
	if tree == nil {
		return
	}
	blankMissing(tree.Root)
}

// blankMissing rewrites every output action below node to pass its value through emptyIfMissing.
func blankMissing(node parse.Node) {
	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return
		}
		for _, child := range n.Nodes {
			blankMissing(child)
		}
	case *parse.ActionNode:
		// An action that only declares variables prints nothing
		if len(n.Pipe.Decl) > 0 {
			return
		}
		n.Pipe.Cmds = append(n.Pipe.Cmds, &parse.CommandNode{
			NodeType: parse.NodeCommand,
			Pos:      n.Pos,
			Args: []parse.Node{
				&parse.IdentifierNode{
					NodeType: parse.NodeIdentifier,
					Pos:      n.Pos,
					Ident:    missingFunc,
				},
			},
		})
	case *parse.IfNode:
		blankMissing(n.List)
		blankMissing(n.ElseList)
	case *parse.RangeNode:
		blankMissing(n.List)
		blankMissing(n.ElseList)
	case *parse.WithNode:
		blankMissing(n.List)
		blankMissing(n.ElseList)
	}
}

// emptyIfMissing returns an empty string for nil and for nil pointers, maps, slices, and interfaces, and any other value unchanged.
func emptyIfMissing(value any) any {
	if value == nil {
		return ""
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Func, reflect.Chan:
		if v.IsNil() {
			return ""
		}
	default:
	}
	return value
}
