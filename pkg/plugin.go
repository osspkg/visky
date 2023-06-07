package pkg

import (
	"github.com/deweppro/goppy/plugins"
	"github.com/osspkg/visky/pkg/images"
	"github.com/osspkg/visky/pkg/markdown"
)

var Plugin = plugins.Plugins{}.Inject(
	images.Plugin,
	markdown.Plugin,
)
