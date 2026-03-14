package main

import (
	"fmt"
	"strings"

	"github.com/sci-ecommerce/issuesherpa/internal/core"
)

func buildIssueFilter(args []string) core.IssueFilter {
	return core.IssueFilter{
		Project:  extractFlag(args, "--project"),
		Source:   strings.ToLower(extractFlag(args, "--source")),
		SortBy:   core.NormalizeSortBy(extractFlag(args, "--sort")),
		SortDesc: hasFlag(args, "--desc"),
		Search:   extractFlag(args, "--search"),
	}
}

func extractFlag(args []string, flag string) string {
	for i := 0; i < len(args); i++ {
		if args[i] != flag {
			continue
		}
		if i+1 < len(args) {
			return args[i+1]
		}
		return ""
	}
	return ""
}

func issueFilterFromArgs(args []string, cmd string) (core.IssueFilter, []string, error) {
	if err := validateFlagValues(args); err != nil {
		return core.IssueFilter{}, nil, err
	}

	if hasFlag(args, "--open") && hasFlag(args, "--resolved") {
		return core.IssueFilter{}, nil, fmt.Errorf("--open and --resolved cannot be used together")
	}

	filter := buildIssueFilter(args)

	if hasFlag(args, "--open") {
		filter.Status = "open"
	}
	if hasFlag(args, "--resolved") {
		filter.Status = "resolved"
	}

	positional := extractPositionalArgs(args)
	if cmd == "search" && filter.Search == "" && len(positional) > 0 {
		filter.Search = strings.Join(positional, " ")
	}

	return filter, positional, nil
}

func validateFlagValues(args []string) error {
	for i := 0; i < len(args); i++ {
		if !isValueFlag(args[i]) {
			continue
		}
		if i+1 >= len(args) {
			return fmt.Errorf("%s requires a value", args[i])
		}
		next := strings.TrimSpace(args[i+1])
		if next == "" || strings.HasPrefix(next, "--") {
			return fmt.Errorf("%s requires a value", args[i])
		}
	}
	return nil
}

func isValueFlag(value string) bool {
	switch value {
	case "--project", "--source", "--sort", "--search":
		return true
	default:
		return false
	}
}

func extractPositionalArgs(args []string) []string {
	var positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			positional = append(positional, a)
			continue
		}
		switch a {
		case "--project", "--source", "--sort", "--search":
			i++
		}
	}
	return positional
}

func parseCSVList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
