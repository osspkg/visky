package hook

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func (*hook) Parse(parent ast.Node, reader text.Reader, pc parser.Context) ast.Node {
	if attrs, ok := parser.ParseAttributes(reader); ok {
		node := &nilInline{
			BaseInline: ast.BaseInline{},
		}
		for _, attr := range attrs {
			var v []byte
			switch vv := attr.Value.(type) {
			case []byte:
				v = vv
			default:
				v = []byte(fmt.Sprintf("%v", vv))
			}
			parent.LastChild().SetAttribute(attr.Name, v)
			node.SetAttribute(attr.Name, v)
		}
		return node
	}
	return nil
}

func (*hook) Trigger() []byte {
	return []byte{'{'}
}
