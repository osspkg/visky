package hook

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/util"
)

var kindNil = ast.NewNodeKind("Nil")

type nilInline struct {
	ast.BaseInline
}

func (a *nilInline) Dump(source []byte, level int) {
	attrs := a.Attributes()
	list := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		var (
			name  = util.BytesToReadOnlyString(attr.Name)
			value = util.BytesToReadOnlyString(util.EscapeHTML(attr.Value.([]byte)))
		)
		list[name] = value
	}
	ast.DumpHelper(a, source, level, list, nil)
}

func (a *nilInline) Kind() ast.NodeKind {
	return kindNil
}
