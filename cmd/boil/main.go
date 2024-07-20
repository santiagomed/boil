package main

func main() {
	if err := cli.rootCmd.Execute(); err != nil {
		panic(err)
	}
}