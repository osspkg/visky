package hook

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type hook struct {
	Unsafe bool
}

func New(unsafe bool) goldmark.Extender {
	return &hook{Unsafe: unsafe}
}

func (v *hook) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(v, 2),
		),
	)
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			util.Prioritized(v, 1),
		),
	)
}
