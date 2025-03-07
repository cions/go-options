// Copyright (c) 2024-2025 cions
// Licensed under the MIT License. See LICENSE for details.

package options_test

import (
	"errors"
	"slices"
	"strconv"
	"testing"

	"github.com/cions/go-options"
)

type OptionCall struct {
	Name     string
	Value    string
	HasValue bool
}

type OptionNCall struct {
	Name   string
	Values []string
}

func (a OptionNCall) Equal(b OptionNCall) bool {
	return a.Name == b.Name && slices.Equal(a.Values, b.Values)
}

type ArgCall struct {
	Index      int
	Value      string
	AfterDDash bool
}

type TestOptions struct {
	OptionHistory  []OptionCall
	OptionNHistory []OptionNCall
	ArgHistory     []ArgCall
	Before         []string
	After          []string
}

func (opts *TestOptions) Kind(name string) options.Kind {
	switch name {
	case "-a", "-b", "-c", "--boolean":
		return options.Boolean
	case "-r", "--required":
		return options.Required
	case "-o", "--optional":
		return options.Optional
	case "-s", "--set":
		return options.TakeTwoArgs
	case "--number":
		return options.Required
	case "--help":
		return options.Boolean
	case "--version":
		return options.Boolean
	default:
		return options.Unknown
	}
}

func (opts *TestOptions) Option(name, value string, hasValue bool) error {
	opts.OptionHistory = append(opts.OptionHistory, OptionCall{
		Name:     name,
		Value:    value,
		HasValue: hasValue,
	})
	switch name {
	case "--number":
		if _, err := strconv.ParseInt(value, 10, strconv.IntSize); err != nil {
			return err
		}
	case "--help":
		return options.ErrHelp
	case "--version":
		return options.ErrVersion
	}
	return nil
}

func (opts *TestOptions) OptionN(name string, values []string) error {
	opts.OptionNHistory = append(opts.OptionNHistory, OptionNCall{
		Name:   name,
		Values: values,
	})
	return nil
}

func (opts *TestOptions) Arg(index int, value string, afterDDash bool) error {
	opts.ArgHistory = append(opts.ArgHistory, ArgCall{
		Index:      index,
		Value:      value,
		AfterDDash: afterDDash,
	})
	return nil
}

func (opts *TestOptions) Args(before, after []string) error {
	if opts.Before != nil || opts.After != nil {
		panic("Args is already called")
	}
	opts.Before = before
	opts.After = after
	return nil
}

func CompareSlice[S ~[]E, E comparable](t *testing.T, name string, actual, expected S) {
	t.Helper()
	if !slices.Equal(actual, expected) {
		t.Errorf("%s: expected %v, but got %v", name, expected, actual)
	}
}

func CompareSliceF[S ~[]E, E interface{ Equal(E) bool }](t *testing.T, name string, actual, expected S) {
	t.Helper()
	if !slices.EqualFunc(actual, expected, func(a, b E) bool { return a.Equal(b) }) {
		t.Errorf("%s: expected %v, but got %v", name, expected, actual)
	}
}

func TestParse(t *testing.T) {
	t.Run("no arguments", func(t *testing.T) {
		opts := &TestOptions{}
		args, err := options.Parse(opts, []string{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		CompareSlice(t, "OptionHistory", opts.OptionHistory, []OptionCall{})
		CompareSliceF(t, "OptionNHistory", opts.OptionNHistory, []OptionNCall{})
		CompareSlice(t, "ArgHistory", opts.ArgHistory, []ArgCall{})
		CompareSlice(t, "Before", opts.Before, []string{})
		CompareSlice(t, "After", opts.After, []string{})
		CompareSlice(t, "Args", args, slices.Concat(opts.Before, opts.After))
	})

	t.Run("parse options", func(t *testing.T) {
		opts := &TestOptions{}
		args, err := options.Parse(opts, []string{
			"-a", "-b", "-r", "val1", "-rval2", "-o", "val3", "-oval4",
			"--boolean", "--required=val5", "--required=", "--required", "val6",
			"--optional=val7", "--optional=", "--optional", "val8", "val9",
			"-s", "name", "value", "-sname", "value", "--set", "name", "value",
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		CompareSlice(t, "OptionHistory", opts.OptionHistory, []OptionCall{
			{Name: "-a"},
			{Name: "-b"},
			{Name: "-r", Value: "val1", HasValue: true},
			{Name: "-r", Value: "val2", HasValue: true},
			{Name: "-o"},
			{Name: "-o", Value: "val4", HasValue: true},
			{Name: "--boolean"},
			{Name: "--required", Value: "val5", HasValue: true},
			{Name: "--required", Value: "", HasValue: true},
			{Name: "--required", Value: "val6", HasValue: true},
			{Name: "--optional", Value: "val7", HasValue: true},
			{Name: "--optional", Value: "", HasValue: true},
			{Name: "--optional", Value: "", HasValue: false},
		})
		CompareSliceF(t, "OptionNHistory", opts.OptionNHistory, []OptionNCall{
			{Name: "-s", Values: []string{"name", "value"}},
			{Name: "-s", Values: []string{"name", "value"}},
			{Name: "--set", Values: []string{"name", "value"}},
		})
		CompareSlice(t, "ArgHistory", opts.ArgHistory, []ArgCall{
			{Index: 0, Value: "val3", AfterDDash: false},
			{Index: 1, Value: "val8", AfterDDash: false},
			{Index: 2, Value: "val9", AfterDDash: false},
		})
		CompareSlice(t, "Before", opts.Before, []string{"val3", "val8", "val9"})
		CompareSlice(t, "After", opts.After, []string{})
		CompareSlice(t, "Args", args, slices.Concat(opts.Before, opts.After))
	})

	t.Run("combined short options", func(t *testing.T) {
		opts := &TestOptions{}
		args, err := options.Parse(opts, []string{
			"-abc", "-abrval1", "-abr", "val2", "-aboval3", "-abo", "val4",
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		CompareSlice(t, "OptionHistory", opts.OptionHistory, []OptionCall{
			{Name: "-a"},
			{Name: "-b"},
			{Name: "-c"},
			{Name: "-a"},
			{Name: "-b"},
			{Name: "-r", Value: "val1", HasValue: true},
			{Name: "-a"},
			{Name: "-b"},
			{Name: "-r", Value: "val2", HasValue: true},
			{Name: "-a"},
			{Name: "-b"},
			{Name: "-o", Value: "val3", HasValue: true},
			{Name: "-a"},
			{Name: "-b"},
			{Name: "-o", Value: "", HasValue: false},
		})
		CompareSliceF(t, "OptionNHistory", opts.OptionNHistory, []OptionNCall{})
		CompareSlice(t, "ArgHistory", opts.ArgHistory, []ArgCall{
			{Index: 0, Value: "val4", AfterDDash: false},
		})
		CompareSlice(t, "Before", opts.Before, []string{"val4"})
		CompareSlice(t, "After", opts.After, []string{})
		CompareSlice(t, "Args", args, slices.Concat(opts.Before, opts.After))
	})

	t.Run("positional arguments", func(t *testing.T) {
		opts := &TestOptions{}
		args, err := options.Parse(opts, []string{
			"-a", "--required", "--", "val1", "-b", "--", "val2", "-a", "--required", "val3", "--", "val4",
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		CompareSlice(t, "OptionHistory", opts.OptionHistory, []OptionCall{
			{Name: "-a"},
			{Name: "--required", Value: "--", HasValue: true},
			{Name: "-b"},
		})
		CompareSliceF(t, "OptionNHistory", opts.OptionNHistory, []OptionNCall{})
		CompareSlice(t, "ArgHistory", opts.ArgHistory, []ArgCall{
			{Index: 0, Value: "val1", AfterDDash: false},
			{Index: 1, Value: "val2", AfterDDash: true},
			{Index: 2, Value: "-a", AfterDDash: true},
			{Index: 3, Value: "--required", AfterDDash: true},
			{Index: 4, Value: "val3", AfterDDash: true},
			{Index: 5, Value: "--", AfterDDash: true},
			{Index: 6, Value: "val4", AfterDDash: true},
		})
		CompareSlice(t, "Before", opts.Before, []string{"val1"})
		CompareSlice(t, "After", opts.After, []string{"val2", "-a", "--required", "val3", "--", "val4"})
		CompareSlice(t, "Args", args, slices.Concat(opts.Before, opts.After))
	})

	t.Run("errors", func(t *testing.T) {
		_, err := options.Parse(&TestOptions{}, []string{"--help"})
		if !errors.Is(err, options.ErrHelp) {
			t.Errorf("expected ErrHelp, but got %#v", err)
		}

		_, err = options.Parse(&TestOptions{}, []string{"--version"})
		if !errors.Is(err, options.ErrVersion) {
			t.Errorf("expected ErrVersion, but got %#v", err)
		}

		_, err = options.Parse(&TestOptions{}, []string{"--number=NaN"})
		if !errors.Is(err, strconv.ErrSyntax) {
			t.Errorf("expected ErrSyntax, but got %#v", err)
		}
		if !errors.Is(err, options.ErrCmdline) {
			t.Errorf("expected ErrCmdline, but got %#v", err)
		}

		_, err = options.Parse(&TestOptions{}, []string{"-r"})
		if !errors.Is(err, options.ErrCmdline) {
			t.Errorf("expected ErrCmdline, but got %#v", err)
		}

		_, err = options.Parse(&TestOptions{}, []string{"--required"})
		if !errors.Is(err, options.ErrCmdline) {
			t.Errorf("expected ErrCmdline, but got %#v", err)
		}

		_, err = options.Parse(&TestOptions{}, []string{"--boolean=true"})
		if !errors.Is(err, options.ErrCmdline) {
			t.Errorf("expected ErrCmdline, but got %#v", err)
		}

		_, err = options.Parse(&TestOptions{}, []string{"--set=name", "value"})
		if !errors.Is(err, options.ErrCmdline) {
			t.Errorf("expected ErrCmdline, but got %#v", err)
		}

		_, err = options.Parse(&TestOptions{}, []string{"--set", "value"})
		if !errors.Is(err, options.ErrCmdline) {
			t.Errorf("expected ErrCmdline, but got %#v", err)
		}

		_, err = options.Parse(&TestOptions{}, []string{"-s", "value"})
		if !errors.Is(err, options.ErrCmdline) {
			t.Errorf("expected ErrCmdline, but got %#v", err)
		}

		_, err = options.Parse(&TestOptions{}, []string{"-svalue"})
		if !errors.Is(err, options.ErrCmdline) {
			t.Errorf("expected ErrCmdline, but got %#v", err)
		}

		_, err = options.Parse(&TestOptions{}, []string{"-x"})
		if !errors.Is(err, options.ErrCmdline) {
			t.Errorf("expected ErrCmdline, but got %#v", err)
		}

		_, err = options.Parse(&TestOptions{}, []string{"-ax"})
		if !errors.Is(err, options.ErrCmdline) {
			t.Errorf("expected ErrCmdline, but got %#v", err)
		}

		_, err = options.Parse(&TestOptions{}, []string{"-xa"})
		if !errors.Is(err, options.ErrCmdline) {
			t.Errorf("expected ErrCmdline, but got %#v", err)
		}

		_, err = options.Parse(&TestOptions{}, []string{"--unknown"})
		if !errors.Is(err, options.ErrCmdline) {
			t.Errorf("expected ErrCmdline, but got %#v", err)
		}

		_, err = options.Parse(&TestOptions{}, []string{"-a-"})
		if !errors.Is(err, options.ErrCmdline) {
			t.Errorf("expected ErrCmdline, but got %#v", err)
		}
	})
}

func TestParsePOSIX(t *testing.T) {
	opts := &TestOptions{}
	args, err := options.ParsePOSIX(opts, []string{
		"-a", "--required", "--", "val1", "-b", "--", "val2", "-a", "--required", "val3", "--", "val4",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	CompareSlice(t, "OptionHistory", opts.OptionHistory, []OptionCall{
		{Name: "-a"},
		{Name: "--required", Value: "--", HasValue: true},
	})
	CompareSliceF(t, "OptionNHistory", opts.OptionNHistory, []OptionNCall{})
	CompareSlice(t, "ArgHistory", opts.ArgHistory, []ArgCall{
		{Index: 0, Value: "val1", AfterDDash: false},
		{Index: 1, Value: "-b", AfterDDash: false},
		{Index: 2, Value: "val2", AfterDDash: true},
		{Index: 3, Value: "-a", AfterDDash: true},
		{Index: 4, Value: "--required", AfterDDash: true},
		{Index: 5, Value: "val3", AfterDDash: true},
		{Index: 6, Value: "--", AfterDDash: true},
		{Index: 7, Value: "val4", AfterDDash: true},
	})
	CompareSlice(t, "Before", opts.Before, []string{
		"val1", "-b",
	})
	CompareSlice(t, "After", opts.After, []string{
		"val2", "-a", "--required", "val3", "--", "val4",
	})
	CompareSlice(t, "Args", args, slices.Concat(opts.Before, opts.After))
}

func TestParseS(t *testing.T) {
	opts := &TestOptions{}
	args, err := options.ParseS(opts, []string{
		"-a", "--required", "--", "val1", "-b", "--", "val2", "-a", "--required", "val3", "--", "val4",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	CompareSlice(t, "OptionHistory", opts.OptionHistory, []OptionCall{
		{Name: "-a"},
		{Name: "--required", Value: "--", HasValue: true},
	})
	CompareSliceF(t, "OptionNHistory", opts.OptionNHistory, []OptionNCall{})
	CompareSlice(t, "ArgHistory", opts.ArgHistory, []ArgCall{
		{Index: 0, Value: "val1", AfterDDash: false},
		{Index: 1, Value: "-b", AfterDDash: false},
		{Index: 2, Value: "--", AfterDDash: false},
		{Index: 3, Value: "val2", AfterDDash: false},
		{Index: 4, Value: "-a", AfterDDash: false},
		{Index: 5, Value: "--required", AfterDDash: false},
		{Index: 6, Value: "val3", AfterDDash: false},
		{Index: 7, Value: "--", AfterDDash: false},
		{Index: 8, Value: "val4", AfterDDash: false},
	})
	CompareSlice(t, "Before", opts.Before, []string{
		"val1", "-b", "--", "val2", "-a", "--required", "val3", "--", "val4",
	})
	CompareSlice(t, "After", opts.After, []string{})
	CompareSlice(t, "Args", args, slices.Concat(opts.Before, opts.After))
}

func TestError(t *testing.T) {
	if !errors.Is(options.ErrHelp, options.ErrCmdline) {
		t.Errorf("ErrHelp should match ErrCmdline")
	}
	if !errors.Is(options.ErrVersion, options.ErrCmdline) {
		t.Errorf("ErrVersion should match ErrCmdline")
	}
	if !errors.Is(options.ErrUnknown, options.ErrCmdline) {
		t.Errorf("ErrUnknown should match ErrCmdline")
	}
	if !errors.Is(options.ErrNoSubcommand, options.ErrCmdline) {
		t.Errorf("ErrNoSubcommand should match ErrCmdline")
	}
	err := options.Errorf("some error")
	if !errors.Is(err, options.ErrCmdline) {
		t.Errorf("Errorf should return ErrCmdline")
	}

	werr := options.Errorf("option -a: %w", strconv.ErrSyntax)
	if !errors.Is(werr, options.ErrCmdline) {
		t.Errorf("Errorf should return ErrCmdline")
	}
	if !errors.Is(werr, strconv.ErrSyntax) {
		t.Errorf("Errorf should wrap an error operand")
	}
}
