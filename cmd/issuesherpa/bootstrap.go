package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sci-ecommerce/issuesherpa/internal/appconfig"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func handleBootstrapCommand(args []string) (bool, int) {
	if len(args) == 0 {
		return false, 0
	}

	switch strings.TrimSpace(args[0]) {
	case "--version", "version":
		fmt.Println(formatVersion())
		return true, 0
	case "init":
		if len(args) > 1 {
			if args[1] == "--help" || args[1] == "-h" {
				printInitHelp()
				return true, 0
			}
			fmt.Fprintln(os.Stderr, "usage: issuesherpa init")
			return true, cliExitCodeUsage
		}
		if err := runInit(); err != nil {
			fmt.Fprintf(os.Stderr, "init failed: %v\n", err)
			return true, 1
		}
		return true, 0
	default:
		return false, 0
	}
}

func formatVersion() string {
	return fmt.Sprintf("issuesherpa %s (commit %s, built %s)", displayVersion(version), commit, buildDate)
}

func runInit() error {
	path, created, err := appconfig.InitDefault()
	if err != nil {
		return err
	}
	if created {
		fmt.Printf("Created config: %s\n", path)
		fmt.Println("Next: edit the config, then run `issuesherpa` or `issuesherpa list`.")
		return nil
	}
	fmt.Printf("Config already exists: %s\n", path)
	return nil
}

func printInitHelp() {
	fmt.Print(`issuesherpa init

Create a user config template at os.UserConfigDir()/issuesherpa/config.toml.
Existing config files are preserved.
`)
}

func displayVersion(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "unknown"
	}
	if v == "dev" || strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}
