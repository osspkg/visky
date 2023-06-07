package hook

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

func (v *hook) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, v.renderImage)
	reg.Register(kindNil, v.renderNil)
}

func (v *hook) renderNil(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (v *hook) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n, ok := node.(*ast.Image)
	if !ok {
		return ast.WalkContinue, nil
	}

	w.WriteString(`<figure><img src="`) //nolint:errcheck
	if v.Unsafe || !html.IsDangerousURL(n.Destination) {
		w.Write(util.EscapeHTML(util.URLEscape(n.Destination, true))) //nolint:errcheck
	}
	w.WriteString(`" alt="`)           //nolint:errcheck
	w.Write(nodeToHTMLText(n, source)) //nolint:errcheck
	w.WriteByte('"')                   //nolint:errcheck
	if n.Attributes() != nil {
		html.RenderAttributes(w, n, html.ImageAttributeFilter)
	}
	w.WriteString(`">`) //nolint:errcheck
	if len(n.Title) > 0 {
		w.WriteString(`<figcaption>`)     //nolint:errcheck
		w.Write(util.EscapeHTML(n.Title)) //nolint:errcheck
		w.WriteString(`</figcaption>`)    //nolint:errcheck
	}
	w.WriteString(`</figure>`) //nolint:errcheck

	return ast.WalkSkipChildren, nil
}
