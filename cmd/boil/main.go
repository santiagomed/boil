package main

import (
	"github.com/santiagomed/boil/internal/cli"
	"github.com/santiagomed/boil/internal/utils"
)

func main() {
	utils.InitLogger()
	cli.Execute()
}
