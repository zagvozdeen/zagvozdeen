package config

type Config struct {
	IsProduction bool
	AppURL       string
}

func New() Config {
	return Config{
		IsProduction: true,
		AppURL:       "http://127.0.0.1:8080",
	}
}
