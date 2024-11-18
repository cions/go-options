// Copyright (c) 2024 cions
// Licensed under the MIT License. See LICENSE for details.

package options_test

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/cions/go-options"
)

type ExampleOptions struct {
	All     bool
	Verbose bool
	File    *os.File
	Number  int
	Color   string
}

func (opts *ExampleOptions) Kind(name string) options.Kind {
	switch name {
	case "-a", "--all":
		return options.Boolean
	case "-v", "--verbose":
		return options.Boolean
	case "-f", "--file":
		return options.Required
	case "-n", "--number":
		return options.Required
	case "--color":
		return options.Optional
	case "-h", "--help":
		return options.Boolean
	case "--version":
		return options.Boolean
	default:
		return options.Unknown
	}
}

func (opts *ExampleOptions) Option(name, value string, hasValue bool) error {
	switch name {
	case "-a", "--all":
		opts.All = true
	case "-v", "--verbose":
		opts.Verbose = true
	case "-f", "--file":
		if value == "-" {
			opts.File = os.Stdin
		} else {
			fh, err := os.Open(value)
			if err != nil {
				return err
			}
			opts.File = fh
		}
	case "-n", "--number":
		parsed, err := strconv.ParseInt(value, 10, strconv.IntSize)
		if err != nil {
			return err
		}
		opts.Number = int(parsed)
	case "--color":
		if !hasValue {
			value = "always"
		}
		switch value {
		case "always", "never", "auto":
			opts.Color = value
		default:
			return options.Errorf("possible values are 'always', 'never', 'auto'")
		}
	case "-h", "--help":
		return options.ErrHelp
	case "--version":
		return options.ErrVersion
	default:
		return options.ErrUnknown
	}
	return nil
}

func Example() {
	opts := &ExampleOptions{
		File:  os.Stdin,
		Color: "auto",
	}

	// args, err := options.Parse(opts, os.Args[1:])
	args, err := options.Parse(opts, []string{"-av", "-n", "42", "--color", "argument"})
	if errors.Is(err, options.ErrHelp) {
		fmt.Println("Usage: example [-av] [-f FILE] [-n NUM] [--color[={always,never,auto}]] [ARGS...]")
		os.Exit(0)
	} else if errors.Is(err, options.ErrVersion) {
		fmt.Println("example 1.0.0")
		os.Exit(0)
	} else if err != nil {
		fmt.Fprintf(os.Stdout, "example: error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("opts.All: %v\n", opts.All)
	fmt.Printf("opts.Verbose: %v\n", opts.Verbose)
	fmt.Printf("opts.File: %v\n", opts.File.Name())
	fmt.Printf("opts.Number: %v\n", opts.Number)
	fmt.Printf("opts.Color: %v\n", opts.Color)
	fmt.Printf("Arguments: %v\n", args)
	// Output:
	// opts.All: true
	// opts.Verbose: true
	// opts.File: /dev/stdin
	// opts.Number: 42
	// opts.Color: always
	// Arguments: [argument]
}
