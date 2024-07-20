package main

import (
	"boil/internal/cli"
	"boil/internal/utils"
)

func main() {
	utils.InitLogger()
	cli.Execute()
}
