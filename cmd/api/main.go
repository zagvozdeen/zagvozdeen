package main

import (
	"github.com/zagvozdeen/zagvozdeen/api"
	"github.com/zagvozdeen/zagvozdeen/config"
)

func main() {
	api.New(config.New()).Run()
}
