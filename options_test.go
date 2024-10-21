// Copyright (c) 2024 cions
// Licensed under the MIT License. See LICENSE for details.

package options

import (
	"errors"
	"slices"
	"strconv"
	"testing"
)

type OptionCall struct {
	Name     string
	Value    string
	HasValue bool
}

type ArgCall struct {
	Index      int
	Value      string
	AfterDDash bool
}

type TestOptions struct {
	OptionHistory []OptionCall
	ArgHistory    []ArgCall
	Before        []string
	After         []string
}

func (opts *TestOptions) Kind(name string) Kind {
	switch name {
	case "-a", "-b", "-c", "--boolean":
		return Boolean
	case "-r", "--required":
		return Required
	case "-o", "--optional":
		return Optional
	case "--number":
		return Required
	case "--help":
		return Boolean
	case "--version":
		return Boolean
	default:
		return Unknown
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
		return ErrHelp
	case "--version":
		return ErrVersion
	}
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
		t.Errorf("%s: expected %v, got %v", name, expected, actual)
	}
}

func TestParse(t *testing.T) {
	t.Run("no arguments", func(t *testing.T) {
		opts := &TestOptions{}
		args, err := Parse(opts, []string{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		CompareSlice(t, "OptionHistory", opts.OptionHistory, []OptionCall{})
		CompareSlice(t, "ArgHistory", opts.ArgHistory, []ArgCall{})
		CompareSlice(t, "Before", opts.Before, []string{})
		CompareSlice(t, "After", opts.After, []string{})
		CompareSlice(t, "Args", args, slices.Concat(opts.Before, opts.After))
	})

	t.Run("parse options", func(t *testing.T) {
		opts := &TestOptions{}
		args, err := Parse(opts, []string{
			"-a", "-b", "-r", "val1", "-rval2", "-o", "val3", "-oval4",
			"--boolean", "--required=val5", "--required=", "--required", "val6",
			"--optional=val7", "--optional=", "--optional", "val8", "val9",
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
		args, err := Parse(opts, []string{
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
		CompareSlice(t, "ArgHistory", opts.ArgHistory, []ArgCall{
			{Index: 0, Value: "val4", AfterDDash: false},
		})
		CompareSlice(t, "Before", opts.Before, []string{"val4"})
		CompareSlice(t, "After", opts.After, []string{})
		CompareSlice(t, "Args", args, slices.Concat(opts.Before, opts.After))
	})

	t.Run("positional arguments", func(t *testing.T) {
		opts := &TestOptions{}
		args, err := Parse(opts, []string{
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
		_, err := Parse(&TestOptions{}, []string{"--help"})
		if !errors.Is(err, ErrHelp) {
			t.Errorf("expected ErrHelp, got %v", err)
		}

		_, err = Parse(&TestOptions{}, []string{"--version"})
		if !errors.Is(err, ErrVersion) {
			t.Errorf("expected ErrVersion, got %v", err)
		}

		_, err = Parse(&TestOptions{}, []string{"--number=NaN"})
		if !errors.Is(err, strconv.ErrSyntax) {
			t.Errorf("expected ErrSyntax, got %v", err)
		}

		_, err = Parse(&TestOptions{}, []string{"-r"})
		if err == nil {
			t.Errorf("expected an error, got nil")
		}

		_, err = Parse(&TestOptions{}, []string{"--required"})
		if err == nil {
			t.Errorf("expected an error, got nil")
		}

		_, err = Parse(&TestOptions{}, []string{"--boolean=true"})
		if err == nil {
			t.Errorf("expected an error, got nil")
		}

		_, err = Parse(&TestOptions{}, []string{"-x"})
		if err == nil {
			t.Errorf("expected an error, got nil")
		}

		_, err = Parse(&TestOptions{}, []string{"-ax"})
		if err == nil {
			t.Errorf("expected an error, got nil")
		}

		_, err = Parse(&TestOptions{}, []string{"-xa"})
		if err == nil {
			t.Errorf("expected an error, got nil")
		}

		_, err = Parse(&TestOptions{}, []string{"--unknown"})
		if err == nil {
			t.Errorf("expected an error, got nil")
		}

		_, err = Parse(&TestOptions{}, []string{"-a-"})
		if err == nil {
			t.Errorf("expected an error, got nil")
		}
	})
}

func TestParsePOSIX(t *testing.T) {
	opts := &TestOptions{}
	args, err := ParsePOSIX(opts, []string{
		"-a", "--required", "--", "val1", "-b", "--", "val2", "-a", "--required", "val3", "--", "val4",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	CompareSlice(t, "OptionHistory", opts.OptionHistory, []OptionCall{
		{Name: "-a"},
		{Name: "--required", Value: "--", HasValue: true},
	})
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
	args, err := ParseS(opts, []string{
		"-a", "--required", "--", "val1", "-b", "--", "val2", "-a", "--required", "val3", "--", "val4",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	CompareSlice(t, "OptionHistory", opts.OptionHistory, []OptionCall{
		{Name: "-a"},
		{Name: "--required", Value: "--", HasValue: true},
	})
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
