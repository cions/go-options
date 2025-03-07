// Copyright (c) 2024-2025 cions
// Licensed under the MIT License. See LICENSE for details.

package options_test

import (
	"errors"
	"fmt"
	"os"

	"github.com/cions/go-options"
)

type ExampleGlobalOptions struct {
	Config  string
	Verbose bool
}

func (opts *ExampleGlobalOptions) Kind(name string) options.Kind {
	switch name {
	case "-c", "--config":
		return options.Required
	case "-v", "--verbose":
		return options.Boolean
	case "-h", "--help":
		return options.Boolean
	case "--version":
		return options.Boolean
	default:
		return options.Unknown
	}
}

func (opts *ExampleGlobalOptions) Option(name, value string, hasValue bool) error {
	switch name {
	case "-c", "--config":
		opts.Config = value
	case "-v", "--verbose":
		opts.Verbose = true
	case "-h", "--help":
		return options.ErrHelp
	case "--version":
		return options.ErrVersion
	default:
		return options.ErrUnknown
	}
	return nil
}

type ExampleRunOptions struct {
	ExampleGlobalOptions
	DryRun  bool
	Files   []string
	Command []string
}

func (opts *ExampleRunOptions) Kind(name string) options.Kind {
	switch name {
	case "-n", "--dry-run":
		return options.Boolean
	default:
		return opts.ExampleGlobalOptions.Kind(name)
	}
}

func (opts *ExampleRunOptions) Option(name, value string, hasValue bool) error {
	switch name {
	case "-n", "--dry-run":
		opts.DryRun = true
	default:
		return opts.ExampleGlobalOptions.Option(name, value, hasValue)
	}
	return nil
}

func (opts *ExampleRunOptions) Args(before, after []string) error {
	if len(after) == 0 {
		return options.ErrHelp
	}
	opts.Files = before
	opts.Command = after
	return nil
}

func ExampleParseS() {
	opts := &ExampleGlobalOptions{
		Config: "example.conf",
	}

	// args, err := options.ParseS(opts, os.Args[1:])
	args, err := options.ParseS(opts, []string{"run", "-v", "file", "--", "cat"})
	switch {
	case errors.Is(err, options.ErrHelp), errors.Is(err, options.ErrNoSubcommand):
		fmt.Println("Usage: example [-c FILE] [-v] run [-n] [FILE...] -- COMMAND [ARGS...]")
		os.Exit(0)
	case errors.Is(err, options.ErrVersion):
		fmt.Println("example 1.0.0")
		os.Exit(0)
	case err != nil:
		fmt.Fprintf(os.Stdout, "example: error: %v\n", err)
		os.Exit(2)
	}

	switch args[0] {
	case "run":
		runopts := &ExampleRunOptions{
			ExampleGlobalOptions: *opts,
		}
		_, err = options.Parse(runopts, args[1:])
		switch {
		case errors.Is(err, options.ErrHelp):
			fmt.Println("Usage: example [-c FILE] [-v] run [-n] [FILE...] -- COMMAND [ARGS...]")
			os.Exit(0)
		case errors.Is(err, options.ErrVersion):
			fmt.Println("example 1.0.0")
			os.Exit(0)
		case err != nil:
			fmt.Fprintf(os.Stdout, "example: error: %v\n", err)
			os.Exit(2)
		}
		fmt.Printf("runopts.Config: %v\n", runopts.Config)
		fmt.Printf("runopts.Verbose: %v\n", runopts.Verbose)
		fmt.Printf("runopts.DryRun: %v\n", runopts.DryRun)
		fmt.Printf("runopts.Files: %v\n", runopts.Files)
		fmt.Printf("runopts.Command: %v\n", runopts.Command)
	default:
		fmt.Fprintf(os.Stdout, "example: error: unknown subcommand %q. See 'example --help'.\n", args[0])
		os.Exit(2)
	}
	// Output:
	// runopts.Config: example.conf
	// runopts.Verbose: true
	// runopts.DryRun: false
	// runopts.Files: [file]
	// runopts.Command: [cat]
}
