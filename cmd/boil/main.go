package main

import (
	"github.com/santiagomed/boil/cli"
	"github.com/santiagomed/boil/pkg/utils"
)

func main() {
	utils.InitLogger()
	cli.Execute()
}
