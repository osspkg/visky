package markdown

type Config struct {
	Markdown ConfigValue `yaml:"markdown"`
}

type ConfigValue struct {
	CJK    bool `yaml:"cjk"`
	Unsafe bool `yaml:"unsafe"`
}
