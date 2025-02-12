package main

import (
	"github.com/zagvozdeen/zagvozdeen/config"
	"github.com/zagvozdeen/zagvozdeen/internal/converter"
)

func main() {
	converter.New(config.New()).Run()
}
