package markdown

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"os"

	d2 "github.com/FurqanSoftware/goldmark-d2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/deweppro/go-sdk/log"
	figure "github.com/mangoumbrella/goldmark-figure"
	"github.com/osspkg/visky/pkg/markdown/hook"
	"github.com/osspkg/visky/pkg/pool"
	fences "github.com/stefanfritsch/goldmark-fences"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/frontmatter"
	"go.abhg.dev/goldmark/hashtag"
	"go.abhg.dev/goldmark/mermaid"
	"go.abhg.dev/goldmark/toc"
	"mvdan.cc/xurls/v2"
	"oss.terrastruct.com/d2/d2layouts/d2dagrelayout"
	"oss.terrastruct.com/d2/d2themes/d2themescatalog"
)

type Markdown struct {
	conf   ConfigValue
	serv   goldmark.Markdown
	ctxOps []parser.ContextOption
}

func New(c ConfigValue) *Markdown {
	return &Markdown{
		conf: c,
		ctxOps: []parser.ContextOption{
			parser.WithIDs(&ids{}),
		},
		serv: goldmark.New(
			goldmark.WithExtensions(
				func(c ConfigValue) []goldmark.Extender {
					ext := make([]goldmark.Extender, 0, 100)
					ext = append(ext,
						hook.New(c.Unsafe),
						extension.Footnote,
						extension.DefinitionList,
						extension.Strikethrough,
						extension.Table,
						extension.TaskList,
						extension.Typographer,
						figure.Figure,
						extension.NewLinkify(
							extension.WithLinkifyAllowedProtocols([][]byte{
								[]byte("http:"),
								[]byte("https:"),
							}),
							extension.WithLinkifyURLRegexp(
								xurls.Strict(),
							),
						),
						meta.New(
							meta.WithStoresInDocument(),
						),
						&d2.Extender{
							// Defaults when omitted
							Layout:  d2dagrelayout.DefaultLayout,
							ThemeID: d2themescatalog.EarthTones.ID,
						},
						&hashtag.Extender{
							Variant:  hashtag.ObsidianVariant,
							Resolver: &hashTag{},
						},
						&fences.Extender{},
						&toc.Extender{
							Title: "Contents",
						},
						&frontmatter.Extender{},
						&mermaid.Extender{
							RenderMode: mermaid.RenderModeServer, // or RenderModeClient
						},
						highlighting.NewHighlighting(
							highlighting.WithStyle("dracula"),
							highlighting.WithFormatOptions(
								chromahtml.WithLineNumbers(true),
							),
						),
					)
					if c.CJK {
						ext = append(ext, extension.CJK)
					}
					return ext
				}(c)...,
			),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
				parser.WithAttribute(),
			),
			goldmark.WithRendererOptions(
				func(c ConfigValue) []renderer.Option {
					options := make([]renderer.Option, 0, 5)
					options = append(options,
						html.WithHardWraps(),
					)
					if c.Unsafe {
						options = append(options, html.WithUnsafe())
					}
					return options
				}(c)...,
			),
		),
	}
}

func (v *Markdown) RenderFile(file string) ([]byte, Meta, error) {
	source, err := os.ReadFile(file)
	if err != nil {
		return nil, Meta{}, fmt.Errorf("read file `%s`: %w", file, err)
	}
	return v.RenderContent(source)
}

func (v *Markdown) RenderContent(source []byte) ([]byte, Meta, error) {
	var b bytes.Buffer
	context := parser.NewContext(v.ctxOps...)
	doc := v.serv.Parser().Parse(text.NewReader(source), parser.WithContext(context))
	if err := v.serv.Renderer().Render(&b, source, doc); err != nil {
		return nil, Meta{}, fmt.Errorf("render markdown: %w", err)
	}

	metaData := Meta{}
	mp := frontmatter.Get(context)
	if err := mp.Decode(&metaData); err != nil {
		return nil, Meta{}, fmt.Errorf("render markdown: %w", err)
	}

	err := ast.Walk(doc, func(node ast.Node, enter bool) (ast.WalkStatus, error) {
		if n, ok := node.(*hashtag.Node); ok && enter {
			metaData.Tags = append(metaData.Tags, string(n.Tag))
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		log.WithFields(log.Fields{
			"err": err.Error(),
		}).Errorf("Markdown meta ast")

	}

	return b.Bytes(), metaData, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type hashTag struct{}

func (v *hashTag) ResolveHashtag(node *hashtag.Node) ([]byte, error) {
	b := pool.ResolveBytes(func(w pool.Writer) {
		w.WriteString("/tags/") //nolint:errcheck
		w.Write(node.Tag)       //nolint:errcheck
		w.WriteString("/")      //nolint:errcheck
	})
	return b, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type ids struct{}

func (v *ids) Generate(value []byte, _ ast.NodeKind) []byte {
	h := fnv.New32a()
	h.Write(value)
	code := hex.EncodeToString(h.Sum(nil))
	return []byte(code)
}

func (v *ids) Put(_ []byte) {}
