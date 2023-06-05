package app

// Config config model
type Config struct {
	Env     string `yaml:"env"`
	PidFile string `yaml:"pid"`
	Level   uint32 `yaml:"level"`
	LogFile string `yaml:"log"`
}
