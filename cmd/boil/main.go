package main

import (
	"boil/internal/cli"
	"boil/internal/utils"
	// tlp "github.com/traceloop/go-openllmetry/traceloop-sdk"
)

func main() {
	utils.InitLogger()
	cli.Execute()
}
