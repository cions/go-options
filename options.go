// Copyright (c) 2024-2025 cions
// Licensed under the MIT License. See LICENSE for details.

// Package options implements command-line option parsing.
package options

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrCmdline is the error returned if the command line is invalid.
	ErrCmdline = errors.New("invalid command line")

	// ErrHelp is the error returned if the user requested to show help message.
	ErrHelp = Errorf("help requested")

	// ErrVersion is the error returned if the user requested to show version information.
	ErrVersion = Errorf("version requested")

	// ErrUnknown is the error returned if an unknown option is provided.
	ErrUnknown = Errorf("unknown option")

	// ErrNoSubcommand is the error returned if no subcommand is provided.
	ErrNoSubcommand = Errorf("no subcommand was provided")
)

type cmdlineError struct{ error }

func (e cmdlineError) Error() string        { return e.error.Error() }
func (e cmdlineError) Unwrap() error        { return errors.Unwrap(e.error) }
func (e cmdlineError) Is(target error) bool { return target == ErrCmdline }

// Errorf wraps fmt.Errorf so that the returned error satisfy errors.Is(err, ErrCmdline).
func Errorf(format string, a ...any) error {
	return cmdlineError{fmt.Errorf(format, a...)}
}

// Kind defines how the option takes arguments.
type Kind int

const (
	Unknown Kind = iota
	Boolean
	Required
	Optional
	TakeTwoArgs
)

// Options is an interface that defines the set of options and stores the parsed result.
type Options interface {
	// Kind is called for each option with name (including dashes) and returns Kind.
	Kind(name string) Kind

	// Option is called for each option with name (including dashes) and value.
	Option(name, value string, hasValue bool) error
}

// OptionsWithOptionN is an interface that adds the OptionN method to Options.
//
// OptionN is called for each TakeTwoArgs option instead of Option.
type OptionsWithOptionN interface {
	Options

	OptionN(name string, values []string) error
}

// OptionsWithArg is an interface that adds the Arg method to Options.
//
// Arg is called for each positional argument, with 0-based index and a boolean
// indicating whether it appears before or after --.
type OptionsWithArg interface {
	Options

	Arg(index int, value string, afterDDash bool) error
}

// OptionsWithArgs is an interface that adds the Args method to Options.
//
// Args is called once at the end, with the positional arguments before and after the --.
type OptionsWithArgs interface {
	Options

	Args(before, after []string) error
}

const (
	earlyExit = 1 << iota
	noDDash
)

func parse(opts Options, args []string, flags int) ([]string, error) {
	var positional []string
	var exited bool

	for len(args) > 0 {
		var name, value string
		var hasValue bool

		switch {
		case args[0] == "--" && flags&noDDash == 0:
			if aopts, ok := opts.(OptionsWithArg); ok {
				for i, arg := range args[1:] {
					if err := aopts.Arg(i+len(positional), arg, true); err != nil {
						return nil, err
					}
				}
			}

			if aopts, ok := opts.(OptionsWithArgs); ok {
				if err := aopts.Args(positional, args[1:]); err != nil {
					return nil, err
				}
			}

			return append(positional, args[1:]...), nil

		case exited, !strings.HasPrefix(args[0], "-"), args[0] == "-", args[0] == "--":
			if aopts, ok := opts.(OptionsWithArg); ok {
				if err := aopts.Arg(len(positional), args[0], false); err != nil {
					return nil, err
				}
			}

			positional = append(positional, args[0])
			args = args[1:]
			if flags&earlyExit != 0 {
				exited = true
			}

			continue

		case strings.HasPrefix(args[0], "--"):
			name, value, hasValue = strings.Cut(args[0], "=")

			switch opts.Kind(name) {
			case Required:
				if !hasValue && len(args) < 2 {
					return nil, Errorf("option %s requires an argument", name)
				}

				if hasValue {
					args = args[1:]
				} else {
					value = args[1]
					hasValue = true
					args = args[2:]
				}

			case Optional:
				args = args[1:]

			case Boolean:
				if hasValue {
					return nil, Errorf("option %s takes no argument", name)
				}

				args = args[1:]

			case TakeTwoArgs:
				if hasValue {
					return nil, Errorf("option %s takes 2 arguments; %s=VALUE form is not permitted", name, name)
				}
				if len(args) < 3 {
					return nil, Errorf("option %s requires 2 arguments", name)
				}

				if nopts, ok := opts.(OptionsWithOptionN); ok {
					if err := nopts.OptionN(name, args[1:3]); err != nil {
						return nil, Errorf("option %s: %w", name, err)
					}
				} else {
					panic("Kind() returned TakeTwoArgs but OptionN method is not implemented")
				}

				args = args[3:]
				continue

			default:
				return nil, Errorf("unknown option %q", name)
			}

		case len(args[0]) > 2:
			name = args[0][:2]

			switch opts.Kind(name) {
			case Required, Optional:
				value = args[0][2:]
				hasValue = true
				args = args[1:]

			case Boolean:
				if args[0][2] == '-' {
					return nil, Errorf("invalid option '-'")
				}

				args[0] = "-" + args[0][2:]

			case TakeTwoArgs:
				if len(args) < 2 {
					return nil, Errorf("option %s requires 2 arguments", name)
				}

				values := []string{args[0][2:], args[1]}
				if nopts, ok := opts.(OptionsWithOptionN); ok {
					if err := nopts.OptionN(name, values); err == ErrUnknown {
						return nil, Errorf("unknown option %q", name)
					} else if err != nil {
						return nil, Errorf("option %s: %w", name, err)
					}
				} else {
					panic("Kind() returned TakeTwoArgs but OptionN method is not implemented")
				}

				args = args[2:]
				continue

			default:
				return nil, Errorf("unknown option %q", name)
			}

		default:
			name = args[0]

			switch opts.Kind(name) {
			case Required:
				if len(args) == 1 {
					return nil, Errorf("option %s requires an argument", name)
				}

				value = args[1]
				hasValue = true
				args = args[2:]

			case Boolean, Optional:
				args = args[1:]

			case TakeTwoArgs:
				if len(args) < 3 {
					return nil, Errorf("option %s requires 2 arguments", name)
				}

				values := []string{args[1], args[2]}
				if nopts, ok := opts.(OptionsWithOptionN); ok {
					if err := nopts.OptionN(name, values); err == ErrUnknown {
						return nil, Errorf("unknown option %q", name)
					} else if err != nil {
						return nil, Errorf("option %s: %w", name, err)
					}
				} else {
					panic("Kind() returns TakeTwoArgs but OptionN method is not implemented")
				}

				args = args[3:]
				continue

			default:
				return nil, Errorf("unknown option %q", name)
			}
		}

		if err := opts.Option(name, value, hasValue); err == ErrUnknown {
			return nil, Errorf("unknown option %q", name)
		} else if err != nil {
			return nil, Errorf("option %s: %w", name, err)
		}
	}

	if aopts, ok := opts.(OptionsWithArgs); ok {
		if err := aopts.Args(positional, nil); err != nil {
			return nil, err
		}
	}

	return positional, nil
}

// Parse parses command-line options from the argument list, which should
// not include the command name. Interleaving of options and non-options is allowed.
// Returns the positional arguments.
func Parse(opts Options, args []string) ([]string, error) {
	return parse(opts, args, 0)
}

// ParsePOSIX parses command-line options from the argument list, which should
// not include the command name. It stop parsing at the first non-option argument.
// Returns the positional arguments.
func ParsePOSIX(opts Options, args []string) ([]string, error) {
	return parse(opts, args, earlyExit)
}

// ParseS parses command-line options from the argument list, which should not
// include the command name. It stop parsing at the first non-option argument
// and does not absorb the first --.
// Returns the positional arguments.
// If no positional arguments was provided, it will return ErrNoSubcommand.
func ParseS(opts Options, args []string) ([]string, error) {
	args, err := parse(opts, args, earlyExit|noDDash)
	if err == nil && len(args) == 0 {
		return nil, ErrNoSubcommand
	}
	return args, err
}
