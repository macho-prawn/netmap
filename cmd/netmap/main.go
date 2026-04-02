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

var newComputeProvider = func(ctx context.Context) (provider.DiscoveryProvider, error) {
	return provider.NewComputeProvider(ctx)
}

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

	files := app.RealFileStore{}

	input, err := app.Validate(files, args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if input.Options.ShowHelp {
		fmt.Fprint(stdout, input.Options.Usage)
		return 0
	}

	discovery, err := newComputeProvider(ctx)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	cli, err := app.New(
		files,
		discovery,
	)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if err := cli.RunValidated(ctx, input); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
