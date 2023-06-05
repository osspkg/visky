package markdown_test

import (
	"reflect"
	"testing"

	"github.com/osspkg/visky/pkg/markdown"
)

func TestUnit_Markdown_RenderContent(t *testing.T) {
	type fields struct {
		conf markdown.ConfigValue
	}
	type args struct {
		source string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		want1   markdown.Meta
		wantErr bool
	}{
		{
			name: "Case 1",
			fields: fields{
				conf: markdown.ConfigValue{
					CJK:    false,
					Unsafe: false,
				},
			},
			args: args{
				source: `---
title: title
summary: summary
tags:
    - markdown
    - goldmark
---

# О нас
Hello #world
` + "```go\nfmt.Println(\"ok\")\nfmt.Println(\"ok\")\n```",
			},
			want: []byte(`<h1>Contents</h1>
<ul>
<li>
<a href="#0b1fb208">О нас</a></li>
</ul>
<h1 id="0b1fb208">О нас</h1>
<p>Hello <span class="hashtag"><a href="/tags/world/">#world</a></span></p>
<pre tabindex="0" style="color:#f8f8f2;background-color:#282a36;"><code><span style="display:flex;"><span style="white-space:pre;user-select:none;margin-right:0.4em;padding:0 0.4em 0 0.4em;color:#7f7f7f">1</span><span>fmt.<span style="color:#50fa7b">Println</span>(<span style="color:#f1fa8c">&#34;ok&#34;</span>)
</span></span><span style="display:flex;"><span style="white-space:pre;user-select:none;margin-right:0.4em;padding:0 0.4em 0 0.4em;color:#7f7f7f">2</span><span>fmt.<span style="color:#50fa7b">Println</span>(<span style="color:#f1fa8c">&#34;ok&#34;</span>)
</span></span></code></pre>`),
			want1: markdown.Meta{
				Title:       "title",
				Description: "",
				Tags:        []string{"markdown", "goldmark", "world"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := markdown.New(tt.fields.conf)
			got, got1, err := v.RenderContent([]byte(tt.args.source))
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RenderContent() content got = `%v`, want `%v`", string(got), string(tt.want))
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("RenderContent() meta \ngot1 = %+v, \nwant = %+v", got1, tt.want1)
			}
		})
	}
}
