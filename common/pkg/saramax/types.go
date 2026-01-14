package saramax

// TODO 待完善的pkg
type Consumer interface {
	Start() error
}

type HandlerConfig struct {
	ConsumeTime int `yaml:"consumeTime"`
	ConsumeNum  int `yaml:"consumeNum"`
}
