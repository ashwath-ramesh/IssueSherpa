package main

import "strings"

func handlePreRuntimeCommand(args []string) (bool, int) {
	args = stripFlags(args, "--offline")
	if len(args) == 0 {
		return false, 0
	}

	switch args[0] {
	case "help", "--help", "-h":
		printCLIHelp()
		return true, 0
	}

	cmd, cmdArgs, options, err := parseCLIInvocation(args)
	if err != nil {
		return true, cliExitCode(runCLI(args, nil))
	}

	if options.Help || options.Describe || shouldFailFastWithoutRuntime(cmd, cmdArgs) {
		return true, cliExitCode(runCLI(args, nil))
	}

	return false, 0
}

func shouldFailFastWithoutRuntime(cmd string, cmdArgs []string) bool {
	switch cmd {
	case "":
		return false
	case "list", "leaderboard":
		_, _, err := issueFilterFromArgs(cmdArgs, cmd)
		return err != nil
	case "json":
		_, _, err := issueFilterFromArgs(cmdArgs, "list")
		return err != nil
	case "search":
		filter, positional, err := issueFilterFromArgs(cmdArgs, cmd)
		if err != nil {
			return true
		}
		return strings.TrimSpace(filter.Search) == "" && len(positional) == 0
	case "show":
		return len(cmdArgs) == 0
	default:
		return true
	}
}
