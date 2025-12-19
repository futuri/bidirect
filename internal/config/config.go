package config

type Config struct {
	InitialSize    int
	KeepAspect     bool
	MinSize        int
	MaxSize        int
	AlphaThreshold uint8
	BorderGrabSize int
	WindowTitle    string
	WSPort         int
}

func DefaultConfig() Config {
	return Config{
		InitialSize:    400,
		KeepAspect:     false,
		MinSize:        100,
		MaxSize:        0,
		AlphaThreshold: 10,
		BorderGrabSize: 8,
		WindowTitle:    "BiDirect",
		WSPort:         8080,
	}
}
