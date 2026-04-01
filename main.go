package main

import (
	"fmt"
	"os"

	"dfl/internal/cli"
)

func main() {
	app := cli.NewApp()
	code, err := app.Run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(code)
}
