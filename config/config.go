package config

type Config struct {
	IsProduction bool
	AppURL       string
}

const isProduction = true

func New() Config {
	url := "http://127.0.0.1:8080"
	if isProduction {
		url = "https://zagvozdeen.ru"
	}
	return Config{
		IsProduction: isProduction,
		AppURL:       url,
	}
}
