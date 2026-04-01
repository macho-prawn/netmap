package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"netmap/internal/app"
	"netmap/internal/provider"
	"netmap/internal/version"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "version" {
		if len(args) != 1 {
			fmt.Fprintln(stderr, "version command does not accept additional arguments")
			return 1
		}
		fmt.Fprintln(stdout, version.Value)
		return 0
	}

	discovery, err := provider.NewComputeProvider(ctx)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	cli, err := app.New(
		app.RealFileStore{},
		discovery,
	)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if err := cli.Run(ctx, args); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
