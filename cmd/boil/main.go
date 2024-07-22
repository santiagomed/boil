package main

import (
	"boil/internal/cli"
	"boil/internal/utils"
	// tlp "github.com/traceloop/go-openllmetry/traceloop-sdk"
)

func main() {
	utils.InitLogger()

	// ctx := context.Background()

	// traceloop, err := tlp.NewClient(ctx, tlp.Config{
	// 	BaseURL: "api-staging.traceloop.com",
	// 	APIKey:  os.Getenv("TRACELOOP_API_KEY"),
	// })
	// defer func() { traceloop.Shutdown(ctx) }()

	// if err != nil {
	// 	fmt.Printf("Traceloop NewClient error: %v\n", err)
	// 	return
	// }

	cli.Execute()
}
