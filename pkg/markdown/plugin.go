package markdown

import (
	"github.com/deweppro/goppy/plugins"
)

var Plugin = plugins.Plugin{
	Config: &Config{},
	Inject: func(c *Config) *Markdown {
		return New(c.Markdown)
	},
}
