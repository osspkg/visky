package markdown

import "time"

type Meta struct {
	Title string `yaml:"title"`

	Cover       string    `yaml:"cover,omitempty"`
	Draft       bool      `yaml:"draft,omitempty"`
	Date        time.Time `yaml:"date,omitempty"`
	Description string    `yaml:"description,omitempty"`
	Lang        string    `yaml:"lang,omitempty"`
	Slug        string    `yaml:"slug,omitempty"`
	Template    string    `yaml:"template,omitempty"`

	Categories []string `yaml:"categories,omitempty"`
	Images     []string `yaml:"images,omitempty"`
	Keywords   []string `yaml:"keywords,omitempty"`
	Tags       []string `yaml:"tags,omitempty"`
}
