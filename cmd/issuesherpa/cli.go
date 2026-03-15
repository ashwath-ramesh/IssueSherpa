package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/sci-ecommerce/issuesherpa/internal/core"
)

type cliOutputMode string

const (
	cliOutputJSON   cliOutputMode = "json"
	cliOutputNDJSON cliOutputMode = "ndjson"
	cliOutputText   cliOutputMode = "text"
)

type cliErrorCode string

const (
	cliErrorCodeInvalidArgument    cliErrorCode = "INVALID_ARGUMENT"
	cliErrorCodeUnknownCommand     cliErrorCode = "UNKNOWN_COMMAND"
	cliErrorCodeMissingArgument    cliErrorCode = "MISSING_ARGUMENT"
	cliErrorCodeNotFoundIssue      cliErrorCode = "ISSUE_NOT_FOUND"
	cliErrorCodeSearchQueryMissing cliErrorCode = "SEARCH_QUERY_REQUIRED"
)

const (
	cliExitCodeUsage    = 2
	cliExitCodeNotFound = 3
)

type cliRunOptions struct {
	Output             cliOutputMode
	Pretty             bool
	Describe           bool
	DescribeDeprecated bool
	Help               bool
	Limit              int
	Offset             int
	HasLimit           bool
	HasOffset          bool
	Fields             []string
}

type cliError struct {
	Code     cliErrorCode `json:"code"`
	Message  string       `json:"message"`
	ExitCode int          `json:"exit_code"`
	Err      error        `json:"-"`
}

type cliErrorContext struct {
	Command string
	Output  cliOutputMode
	Pretty  bool
}

func (e *cliError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil && strings.TrimSpace(e.Err.Error()) != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Err.Error())
	}
	return e.Message
}

type cliSchema struct {
	Version       string             `json:"version"`
	Name          string             `json:"name"`
	Description   string             `json:"description"`
	Usage         string             `json:"usage,omitempty"`
	OutputModes   []string           `json:"output_modes,omitempty"`
	Response      *cliSchemaResponse `json:"response,omitempty"`
	Error         *cliSchemaResponse `json:"error,omitempty"`
	Commands      []cliSchemaCommand `json:"commands,omitempty"`
	Command       string             `json:"command,omitempty"`
	Fields        []cliSchemaField   `json:"fields,omitempty"`
	Options       []cliSchemaOption  `json:"options,omitempty"`
	Examples      []string           `json:"examples,omitempty"`
	GlobalOptions []cliSchemaOption  `json:"global_options"`
}

type cliSchemaResponse struct {
	Type   string           `json:"type"`
	Fields []cliSchemaField `json:"fields"`
}

type cliSchemaField struct {
	Name        string           `json:"name"`
	Type        string           `json:"type"`
	Description string           `json:"description,omitempty"`
	Fields      []cliSchemaField `json:"fields,omitempty"`
	Items       *cliSchemaField  `json:"items,omitempty"`
}

type cliSchemaCommand struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Deprecated  bool               `json:"deprecated,omitempty"`
	Usage       string             `json:"usage"`
	OutputModes []string           `json:"output_modes"`
	Fields      []cliSchemaField   `json:"fields"`
	Options     []cliSchemaOption  `json:"options"`
	Examples    []string           `json:"examples"`
	Response    *cliSchemaResponse `json:"response,omitempty"`
	Error       *cliSchemaResponse `json:"error,omitempty"`
}

type cliLeaderboardRow struct {
	Rank     int
	Reporter string
	Count    int
	Percent  float64
	Bar      string
}

type cliSchemaOption struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Default     string   `json:"default,omitempty"`
	Values      []string `json:"values,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Deprecated  bool     `json:"deprecated,omitempty"`
}

const cliSchemaVersion = "1.0"

var cliIssueSchemaFields = []cliSchemaField{
	{Name: "id", Type: "string", Description: "Issue unique identifier"},
	{Name: "short_id", Type: "string", Description: "Issue short identifier"},
	{Name: "title", Type: "string", Description: "Issue title"},
	{Name: "status", Type: "string", Description: "Issue status"},
	{Name: "project", Type: "object", Description: "Project object"},
	{Name: "project_id", Type: "string", Description: "Project ID"},
	{Name: "project_slug", Type: "string", Description: "Project slug"},
	{Name: "project_name", Type: "string", Description: "Project name"},
	{Name: "reporter", Type: "string", Description: "Issue reporter"},
	{Name: "url", Type: "string", Description: "Issue URL"},
	{Name: "source", Type: "string", Description: "Issue source"},
	{Name: "events", Type: "string", Description: "Event count"},
	{Name: "users", Type: "integer", Description: "Number of unique users"},
	{Name: "first_seen", Type: "string", Description: "ISO8601 first seen"},
	{Name: "last_seen", Type: "string", Description: "ISO8601 last seen"},
	{Name: "assigned_to", Type: "object", Description: "Assignee"},
}

var cliLeaderboardSchemaFields = []cliSchemaField{
	{Name: "rank", Type: "integer", Description: "Position in leaderboard"},
	{Name: "reporter", Type: "string", Description: "Reporter name"},
	{Name: "count", Type: "integer", Description: "Issue count"},
	{Name: "percent", Type: "number", Description: "Percent of total issues"},
	{Name: "bar", Type: "string", Description: "Text bar representation"},
}

var cliErrorSchemaFields = []cliSchemaField{
	{Name: "command", Type: "string", Description: "Command that failed"},
	{
		Name: "error",
		Type: "object",
		Fields: []cliSchemaField{
			{Name: "code", Type: "string", Description: "Stable machine code"},
			{Name: "message", Type: "string", Description: "Machine-readable message"},
			{Name: "exit_code", Type: "integer", Description: "Suggested exit code"},
		},
	},
}

var cliCommandErrorResponse = &cliSchemaResponse{
	Type:   "object",
	Fields: cliErrorSchemaFields,
}

func commandResponseSchema(name string) *cliSchemaResponse {
	switch name {
	case "show":
		return &cliSchemaResponse{
			Type: "object",
			Fields: []cliSchemaField{
				{Name: "command", Type: "string", Description: "Command name"},
				{Name: "count", Type: "integer", Description: "Rows returned"},
				{Name: "total", Type: "integer", Description: "Total matches"},
				{Name: "offset", Type: "integer", Description: "Offset applied"},
				{Name: "data", Type: "array", Items: &cliSchemaField{
					Type:        "object",
					Description: "Single issue result (single-element array).",
					Fields:      cliIssueSchemaFields,
				}},
			},
		}
	case "leaderboard":
		return &cliSchemaResponse{
			Type: "object",
			Fields: []cliSchemaField{
				{Name: "command", Type: "string", Description: "Command name"},
				{Name: "count", Type: "integer", Description: "Rows returned"},
				{Name: "total", Type: "integer", Description: "Total rows before pagination"},
				{Name: "offset", Type: "integer", Description: "Offset applied"},
				{Name: "data", Type: "array", Items: &cliSchemaField{Type: "object", Fields: cliLeaderboardSchemaFields}},
			},
		}
	default:
		return &cliSchemaResponse{
			Type: "object",
			Fields: []cliSchemaField{
				{Name: "command", Type: "string", Description: "Command name"},
				{Name: "count", Type: "integer", Description: "Rows returned"},
				{Name: "total", Type: "integer", Description: "Total matches"},
				{Name: "offset", Type: "integer", Description: "Offset applied"},
				{Name: "data", Type: "array", Items: &cliSchemaField{Type: "object", Fields: cliIssueSchemaFields}},
				{Name: "limit", Type: "integer", Description: "Limit applied"},
			},
		}
	}
}

var cliCommandSchema = map[string]cliSchemaCommand{
	"list": {
		Name:        "list",
		Description: "List issues with optional filters.",
		Usage:       "issuesherpa list [--project <slug>] [--source <sentry|gitlab|github>] [--search <query>] [--open|--resolved] [--sort <created|updated|project|reporter|status|title|source|id>] [--desc]",
		OutputModes: []string{"json", "ndjson", "text"},
		Fields:      cliIssueSchemaFields,
		Response:    commandResponseSchema("list"),
		Error:       cliCommandErrorResponse,
		Options: []cliSchemaOption{
			{Name: "--project", Type: "string", Description: "Filter by project/repo slug"},
			{Name: "--source", Type: "string", Description: "Filter by source"},
			{Name: "--search", Type: "string", Description: "Search query"},
			{Name: "--open", Type: "boolean", Description: "Only open issues"},
			{Name: "--resolved", Type: "boolean", Description: "Only resolved issues"},
			{Name: "--sort", Type: "string", Description: "Sort field"},
			{Name: "--desc", Type: "boolean", Description: "Descending sort"},
		},
		Examples: []string{
			"issuesherpa list --open --source github",
			"issuesherpa list --project frontend --sort updated --desc",
		},
	},
	"search": {
		Name:        "search",
		Description: "Search issues by query.",
		Usage:       "issuesherpa search <query> [--project <slug>] [--source <sentry|gitlab|github>] [--open|--resolved] [--sort ...] [--desc]",
		OutputModes: []string{"json", "ndjson", "text"},
		Fields:      cliIssueSchemaFields,
		Response:    commandResponseSchema("search"),
		Error:       cliCommandErrorResponse,
		Options: []cliSchemaOption{
			{Name: "--search", Type: "string", Description: "Search query. If omitted, positional args are joined as the query."},
			{Name: "--project", Type: "string", Description: "Filter by project/repo slug"},
			{Name: "--source", Type: "string", Description: "Filter by source"},
			{Name: "--open", Type: "boolean", Description: "Only open issues"},
			{Name: "--resolved", Type: "boolean", Description: "Only resolved issues"},
			{Name: "--sort", Type: "string", Description: "Sort field"},
			{Name: "--desc", Type: "boolean", Description: "Descending sort"},
		},
		Examples: []string{
			"issuesherpa search \"timeout\" --source sentry",
			"issuesherpa search --search \"error\" --open",
		},
	},
	"show": {
		Name:        "show",
		Description: "Show one issue by id or short id.",
		Usage:       "issuesherpa show <ISSUE-ID>",
		OutputModes: []string{"json", "ndjson", "text"},
		Fields:      cliIssueSchemaFields,
		Response:    commandResponseSchema("show"),
		Error:       cliCommandErrorResponse,
		Options: []cliSchemaOption{
			{Name: "ISSUE-ID", Type: "string", Required: true},
		},
		Examples: []string{
			"issuesherpa show github:100",
			"issuesherpa show org/repo#10",
		},
	},
	"leaderboard": {
		Name:        "leaderboard",
		Description: "Show issue count by reporter.",
		Usage:       "issuesherpa leaderboard [--project <slug>] [--source <sentry|gitlab|github>] [--open|--resolved] [--sort ...] [--desc]",
		OutputModes: []string{"json", "ndjson", "text"},
		Fields:      cliLeaderboardSchemaFields,
		Response:    commandResponseSchema("leaderboard"),
		Error:       cliCommandErrorResponse,
		Options: []cliSchemaOption{
			{Name: "--project", Type: "string", Description: "Filter by project/repo slug"},
			{Name: "--source", Type: "string", Description: "Filter by source"},
			{Name: "--open", Type: "boolean", Description: "Only open issues"},
			{Name: "--resolved", Type: "boolean", Description: "Only resolved issues"},
		},
		Examples: []string{
			"issuesherpa leaderboard --project frontend",
			"issuesherpa leaderboard --resolved",
		},
	},
	"json": {
		Name:        "json",
		Description: "Deprecated alias for list JSON output.",
		Deprecated:  true,
		Usage:       "issuesherpa json [--project <slug>] [--source <sentry|gitlab|github>] [--search <query>] [--open|--resolved] [--sort ...] [--desc]",
		OutputModes: []string{"json"},
		Fields:      cliIssueSchemaFields,
		Response:    commandResponseSchema("json"),
		Error:       cliCommandErrorResponse,
		Options: []cliSchemaOption{
			{Name: "--project", Type: "string", Description: "Filter by project/repo slug"},
			{Name: "--source", Type: "string", Description: "Filter by source"},
			{Name: "--search", Type: "string", Description: "Search query"},
			{Name: "--open", Type: "boolean", Description: "Only open issues"},
			{Name: "--resolved", Type: "boolean", Description: "Only resolved issues"},
			{Name: "--sort", Type: "string", Description: "Sort field"},
			{Name: "--desc", Type: "boolean", Description: "Descending sort"},
		},
		Examples: []string{
			"issuesherpa json --project frontend",
			"issuesherpa json --open --search timeout",
		},
	},
}

var cliGlobalSchemaOptions = []cliSchemaOption{
	{Name: "--schema", Type: "boolean", Description: "Print machine-readable command metadata."},
	{Name: "--describe", Type: "boolean", Description: "Deprecated alias for --schema.", Deprecated: true},
	{Name: "--output", Type: "string", Values: []string{"json", "ndjson", "text"}, Default: "json", Description: "Output format. Can be set with ISSUESHERPA_OUTPUT."},
	{Name: "--pretty", Type: "boolean", Description: "Pretty-print JSON output (no effect for text)."},
	{Name: "--fields", Type: "string", Description: "Comma-separated JSON fields"},
	{Name: "--limit", Type: "integer", Description: "Return at most N entries"},
	{Name: "--offset", Type: "integer", Description: "Skip first N entries"},
	{Name: "--help", Type: "boolean"},
	{Name: "-h", Type: "boolean", Description: "Show usage"},
}

func runCLI(args []string, issues []Issue) error {
	if len(args) == 0 {
		printCLIHelp()
		return nil
	}

	cmd, cmdArgs, options, err := parseCLIInvocation(args)
	if err != nil {
		return writeCLIError("", err, cliErrorCodeInvalidArgument, cliExitCodeUsage, cliErrorContext{
			Command: "issuesherpa",
			Output:  cliOutputJSON,
			Pretty:  true,
		})
	}

	if options.Help {
		if cmd == "" {
			printCLIHelp()
			return nil
		}
		printCLICommandHelp(cmd)
		return nil
	}

	if options.Describe {
		if options.DescribeDeprecated {
			fmt.Fprintln(os.Stderr, "warning: --describe is deprecated, use --schema")
		}
		if err := writeCLISchema(cmd, options); err != nil {
			return writeCLIError(err.Error(), err, cliErrorCodeInvalidArgument, cliExitCodeUsage, cliErrorContext{
				Command: cmd,
				Output:  cliOutputJSON,
				Pretty:  true,
			})
		}
		return nil
	}

	switch cmd {
	case "list":
		filter, positional, err := issueFilterFromArgs(cmdArgs, cmd)
		if err != nil {
			return writeCLIError("Invalid filter options", err, cliErrorCodeInvalidArgument, cliExitCodeUsage, cliErrorContext{
				Command: cmd,
				Output:  options.Output,
				Pretty:  options.Pretty,
			})
		}
		if strings.TrimSpace(filter.Search) == "" && len(positional) > 0 {
			filter.Search = strings.Join(positional, " ")
		}
		return outputIssueCollection(cmd, core.ApplyFilters(issues, filter), options)
	case "search":
		filter, positional, err := issueFilterFromArgs(cmdArgs, cmd)
		if err != nil {
			return writeCLIError("Invalid filter options", err, cliErrorCodeInvalidArgument, cliExitCodeUsage, cliErrorContext{
				Command: cmd,
				Output:  options.Output,
				Pretty:  options.Pretty,
			})
		}
		if strings.TrimSpace(filter.Search) == "" && len(positional) > 0 {
			filter.Search = strings.Join(positional, " ")
		}
		if strings.TrimSpace(filter.Search) == "" {
			return writeCLIError(
				"Search query is required. Pass query as positional arguments or --search.",
				fmt.Errorf("search query required"),
				cliErrorCodeSearchQueryMissing,
				cliExitCodeUsage,
				cliErrorContext{
					Command: cmd,
					Output:  options.Output,
					Pretty:  options.Pretty,
				},
			)
		}
		return outputIssueCollection(cmd, core.ApplyFilters(issues, filter), options)
	case "show":
		if len(cmdArgs) == 0 {
			return writeCLIError(
				"Issue id is required.",
				fmt.Errorf("issue id required"),
				cliErrorCodeMissingArgument,
				cliExitCodeUsage,
				cliErrorContext{
					Command: cmd,
					Output:  options.Output,
					Pretty:  options.Pretty,
				},
			)
		}
		issue := core.FindIssue(issues, cmdArgs[0])
		if issue == nil {
			return writeCLIError(fmt.Sprintf("Issue %s not found", cmdArgs[0]), fmt.Errorf("issue not found"),
				cliErrorCodeNotFoundIssue,
				cliExitCodeNotFound,
				cliErrorContext{
					Command: cmd,
					Output:  options.Output,
					Pretty:  options.Pretty,
				})
		}
		return outputSingleIssue(*issue, options)
	case "leaderboard":
		filter, _, err := issueFilterFromArgs(cmdArgs, cmd)
		if err != nil {
			return writeCLIError("Invalid filter options", err, cliErrorCodeInvalidArgument, cliExitCodeUsage, cliErrorContext{
				Command: cmd,
				Output:  options.Output,
				Pretty:  options.Pretty,
			})
		}
		filtered := core.ApplyFilters(issues, filter)
		return outputLeaderboard(filtered, options)
	case "json":
		filter, positional, err := issueFilterFromArgs(cmdArgs, "list")
		if err != nil {
			return writeCLIError("Invalid filter options", err, cliErrorCodeInvalidArgument, cliExitCodeUsage, cliErrorContext{
				Command: cmd,
				Output:  options.Output,
				Pretty:  options.Pretty,
			})
		}
		if strings.TrimSpace(filter.Search) == "" && len(positional) > 0 {
			filter.Search = strings.Join(positional, " ")
		}
		options.Output = cliOutputJSON
		return outputIssueCollection(cmd, core.ApplyFilters(issues, filter), options)
	default:
		return writeCLIError(fmt.Sprintf("unknown command: %s", cmd), fmt.Errorf("unknown command"), cliErrorCodeUnknownCommand, cliExitCodeUsage, cliErrorContext{
			Command: cmd,
			Output:  options.Output,
			Pretty:  options.Pretty,
		})
	}
}

func parseCLIInvocation(args []string) (string, []string, cliRunOptions, error) {
	defaultOutput := cliOutputJSON
	if envOutput := strings.TrimSpace(os.Getenv("ISSUESHERPA_OUTPUT")); envOutput != "" {
		tmp := cliRunOptions{Output: defaultOutput}
		if err := assignGlobalValueOption("--output", envOutput, &tmp); err != nil {
			return "", nil, cliRunOptions{}, err
		}
		defaultOutput = tmp.Output
	}

	options := cliRunOptions{
		Output: defaultOutput,
		Limit:  0,
		Offset: 0,
		Fields: nil,
	}
	if isTruthyEnv(os.Getenv("ISSUESHERPA_PRETTY")) {
		options.Pretty = true
	}

	cmd := ""
	cmdArgs := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		current := args[i]
		handled, skip, err := parseGlobalFlag(current, args, i, cmd != "", &options)
		if err != nil {
			return "", nil, cliRunOptions{}, err
		}
		if handled {
			i += skip
			continue
		}
		if cmd == "" && !strings.HasPrefix(current, "--") {
			cmd = current
			continue
		}
		cmdArgs = append(cmdArgs, current)
	}

	return cmd, cmdArgs, options, nil
}

func parseGlobalFlag(name string, args []string, idx int, rejectUnknownFlags bool, options *cliRunOptions) (bool, int, error) {
	if isGlobalBoolFlag(name) {
		switch name {
		case "--schema":
			options.Describe = true
		case "--describe":
			options.Describe = true
			options.DescribeDeprecated = true
		case "--help", "-h":
			options.Help = true
		case "--pretty":
			options.Pretty = true
		}
		return true, 0, nil
	}

	if isGlobalValueFlag(name) {
		if idx+1 >= len(args) {
			return true, 0, fmt.Errorf("%s requires a value", name)
		}
		next := strings.TrimSpace(args[idx+1])
		if next == "" || strings.HasPrefix(next, "--") {
			return true, 0, fmt.Errorf("%s requires a value", name)
		}
		if err := assignGlobalValueOption(name, next, options); err != nil {
			return true, 0, err
		}
		return true, 1, nil
	}

	if rejectUnknownFlags {
		if strings.HasPrefix(name, "--") {
			return false, 0, fmt.Errorf("unknown global option: %s", name)
		}
		return false, 0, nil
	}

	return false, 0, nil
}

func isTruthyEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on", "y":
		return true
	default:
		return false
	}
}

func isGlobalBoolFlag(name string) bool {
	switch name {
	case "--schema", "--describe", "--pretty", "--help", "-h":
		return true
	default:
		return false
	}
}

func isGlobalValueFlag(name string) bool {
	switch name {
	case "--output", "--fields", "--limit", "--offset":
		return true
	default:
		return false
	}
}

func assignGlobalValueOption(name, value string, options *cliRunOptions) error {
	switch name {
	case "--output":
		switch strings.ToLower(value) {
		case string(cliOutputJSON):
			options.Output = cliOutputJSON
		case string(cliOutputNDJSON):
			options.Output = cliOutputNDJSON
		case string(cliOutputText):
			options.Output = cliOutputText
		default:
			return fmt.Errorf("unknown output mode: %s", value)
		}
	case "--fields":
		options.Fields = parseCSVList(value)
	case "--limit":
		limit, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid --limit value: %s", value)
		}
		if limit < 0 {
			return fmt.Errorf("--limit must be >= 0")
		}
		options.Limit = limit
		options.HasLimit = true
	case "--offset":
		offset, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid --offset value: %s", value)
		}
		if offset < 0 {
			return fmt.Errorf("--offset must be >= 0")
		}
		options.Offset = offset
		options.HasOffset = true
	default:
		return fmt.Errorf("unsupported option: %s", name)
	}
	return nil
}

func writeCLISchema(cmd string, options cliRunOptions) error {
	if cmd == "" {
		commands := make([]cliSchemaCommand, 0, len(cliCommandSchema))
		for _, key := range sortedCommandKeys() {
			commands = append(commands, cliCommandSchema[key])
		}
		return writeJSONOutput(cliSchema{
			Version:       cliSchemaVersion,
			Name:          "issuesherpa",
			Description:   "IssueSherpa command-line interface",
			Error:         cliCommandErrorResponse,
			Commands:      commands,
			GlobalOptions: cliGlobalSchemaOptions,
		}, options)
	}
	command, ok := cliCommandSchema[cmd]
	if !ok {
		return fmt.Errorf("unknown command: %s", cmd)
	}
	return writeJSONOutput(cliSchema{
		Version:       cliSchemaVersion,
		Name:          "issuesherpa",
		Command:       cmd,
		Description:   command.Description,
		Usage:         command.Usage,
		OutputModes:   command.OutputModes,
		Fields:        command.Fields,
		Options:       command.Options,
		Examples:      command.Examples,
		Response:      command.Response,
		Error:         command.Error,
		GlobalOptions: cliGlobalSchemaOptions,
	}, options)
}

func sortedCommandKeys() []string {
	keys := make([]string, 0, len(cliCommandSchema))
	for key := range cliCommandSchema {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func writeJSONOutput(payload any, options cliRunOptions) error {
	enc := json.NewEncoder(os.Stdout)
	if options.Pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(payload); err != nil {
		return fmt.Errorf("error writing JSON output: %w", err)
	}
	return nil
}

func writeCLIError(message string, err error, code cliErrorCode, exitCode int, context cliErrorContext) error {
	if message == "" {
		message = err.Error()
	}
	if context.Output == cliOutputJSON || context.Output == cliOutputNDJSON {
		if encodeErr := writeJSONOutput(map[string]any{
			"command": context.Command,
			"error": map[string]any{
				"code":      string(code),
				"message":   message,
				"exit_code": exitCode,
			},
		}, cliRunOptions{Pretty: context.Pretty}); encodeErr != nil {
			fmt.Fprintf(os.Stderr, "%v\n", encodeErr)
			return &cliError{Code: code, Message: message, ExitCode: exitCode, Err: err}
		}
		return &cliError{Code: code, Message: message, ExitCode: exitCode, Err: err}
	}
	fmt.Fprintln(os.Stderr, message)
	return &cliError{Code: code, Message: message, ExitCode: exitCode, Err: err}
}

func outputIssueCollection(command string, issues []Issue, options cliRunOptions) error {
	total := len(issues)
	issues = paginateItems(issues, options.Offset, options.Limit, options.HasOffset, options.HasLimit)

	if options.Output == cliOutputText {
		cliListIssuesText(issues, total, total > 0 && command == "search")
		return nil
	}
	if options.Output == cliOutputNDJSON {
		return writeNDJSONIssueCollection(command, issues, total, options)
	}

	out := make([]map[string]any, 0, len(issues))
	for _, issue := range issues {
		out = append(out, projectIssueMap(issue, options.Fields))
	}
	payload := map[string]any{
		"command": command,
		"count":   len(out),
		"total":   total,
		"offset":  options.Offset,
		"data":    out,
	}
	if options.HasLimit {
		payload["limit"] = options.Limit
	}
	return writeJSONOutput(payload, options)
}

func outputSingleIssue(issue Issue, options cliRunOptions) error {
	if options.Output == cliOutputText {
		cliShowIssueText(issue)
		return nil
	}
	if options.Output == cliOutputNDJSON {
		return writeNDJSONSingleIssue(issue, options)
	}
	return writeJSONOutput(map[string]any{
		"command": "show",
		"count":   1,
		"total":   1,
		"offset":  options.Offset,
		"data":    []map[string]any{projectIssueMap(issue, options.Fields)},
	}, options)
}

func outputLeaderboard(issues []Issue, options cliRunOptions) error {
	raw := core.BuildLeaderboard(issues)
	totalEntries := len(raw)
	totalIssues := len(issues)

	rows := make([]cliLeaderboardRow, 0, totalEntries)
	maxCount := 0
	if len(raw) > 0 {
		maxCount = raw[0].Count
	}
	for i, entry := range raw {
		pct := 0.0
		if totalIssues > 0 {
			pct = (float64(entry.Count) / float64(totalIssues)) * 100
			pct = math.Round(pct*10) / 10
		}
		barLen := 0
		if maxCount > 0 {
			barLen = (entry.Count * 30) / maxCount
		}
		rows = append(rows, cliLeaderboardRow{
			Rank:     i + 1,
			Reporter: sanitizeTerminalText(entry.Name),
			Count:    entry.Count,
			Percent:  pct,
			Bar:      strings.Repeat("#", barLen),
		})
	}

	rows = paginateItems(rows, options.Offset, options.Limit, options.HasOffset, options.HasLimit)

	if options.Output == cliOutputText {
		cliLeaderboardText(rows, len(raw), totalIssues)
		return nil
	}
	if options.Output == cliOutputNDJSON {
		return writeNDJSONLeaderboardCollection(rows, len(raw), options)
	}

	payloadRows := make([]map[string]any, 0, len(rows))
	for _, item := range rows {
		payloadRows = append(payloadRows, applyFieldSelection(map[string]any{
			"rank":     item.Rank,
			"reporter": item.Reporter,
			"count":    item.Count,
			"percent":  item.Percent,
			"bar":      item.Bar,
		}, options.Fields))
	}

	payload := map[string]any{
		"command": "leaderboard",
		"count":   len(payloadRows),
		"total":   totalEntries,
		"offset":  options.Offset,
		"data":    payloadRows,
	}
	if options.HasLimit {
		payload["limit"] = options.Limit
	}
	return writeJSONOutput(payload, options)
}

func paginateItems[T any](items []T, offset int, limit int, hasOffset bool, hasLimit bool) []T {
	if len(items) == 0 {
		return items
	}
	if !hasOffset || offset < 0 {
		offset = 0
	}
	if offset > len(items) {
		return nil
	}
	end := len(items)
	if hasLimit && limit >= 0 && offset+limit < end {
		end = offset + limit
	}
	return items[offset:end]
}

func writeNDJSONSingleIssue(issue Issue, options cliRunOptions) error {
	payload := map[string]any{
		"command": "show",
		"count":   1,
		"total":   1,
		"offset":  options.Offset,
		"data":    projectIssueMap(issue, options.Fields),
	}
	if options.HasLimit {
		payload["limit"] = options.Limit
	}
	return writeJSONLine(payload)
}

func writeNDJSONIssueCollection(command string, issues []Issue, total int, options cliRunOptions) error {
	for _, issue := range issues {
		payload := map[string]any{
			"command": command,
			"count":   1,
			"total":   total,
			"offset":  options.Offset,
			"data":    projectIssueMap(issue, options.Fields),
		}
		if options.HasLimit {
			payload["limit"] = options.Limit
		}
		if err := writeJSONLine(payload); err != nil {
			return err
		}
	}
	return nil
}

func writeJSONLine(payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error writing NDJSON output: %w", err)
	}
	if _, err := fmt.Fprintln(os.Stdout, string(data)); err != nil {
		return err
	}
	return nil
}

func writeNDJSONLeaderboardCollection(rows []cliLeaderboardRow, totalEntries int, options cliRunOptions) error {
	for _, row := range rows {
		payload := map[string]any{
			"command": "leaderboard",
			"count":   1,
			"total":   totalEntries,
			"offset":  options.Offset,
			"data": applyFieldSelection(map[string]any{
				"rank":     row.Rank,
				"reporter": row.Reporter,
				"count":    row.Count,
				"percent":  row.Percent,
				"bar":      row.Bar,
			}, options.Fields),
		}
		if options.HasLimit {
			payload["limit"] = options.Limit
		}
		if err := writeJSONLine(payload); err != nil {
			return err
		}
	}
	return nil
}

func projectIssueMap(issue Issue, fields []string) map[string]any {
	payload := map[string]any{
		"id":           sanitizeTerminalText(issue.ID),
		"short_id":     sanitizeTerminalText(issue.ShortID),
		"title":        sanitizeTerminalText(issue.Title),
		"status":       sanitizeTerminalText(issue.Status),
		"reporter":     sanitizeTerminalText(issue.Reporter),
		"source":       sanitizeTerminalText(issue.Source),
		"url":          sanitizeTerminalText(issue.URL),
		"events":       sanitizeTerminalText(issue.Count),
		"users":        issue.UserCount,
		"first_seen":   issue.FirstSeen,
		"last_seen":    issue.LastSeen,
		"project_id":   issue.Project.ID,
		"project_slug": sanitizeTerminalText(issue.Project.Slug),
		"project_name": sanitizeTerminalText(issue.Project.Name),
		"project": map[string]any{
			"id":   issue.Project.ID,
			"slug": sanitizeTerminalText(issue.Project.Slug),
			"name": sanitizeTerminalText(issue.Project.Name),
		},
	}
	if issue.AssignedTo != nil {
		payload["assigned_to"] = map[string]any{
			"name":  sanitizeTerminalText(issue.AssignedTo.Name),
			"email": sanitizeTerminalText(issue.AssignedTo.Email),
		}
	}

	if len(fields) == 0 {
		return payload
	}
	return applyFieldSelection(payload, fields)
}

func applyFieldSelection(payload map[string]any, fields []string) map[string]any {
	if len(fields) == 0 {
		return payload
	}
	selected := make(map[string]any, len(fields))
	for _, field := range fields {
		key := normalizeFieldName(field)
		if value, ok := payload[key]; ok {
			selected[key] = value
		}
	}
	return selected
}

func normalizeFieldName(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(value, "-", "_")))
}

func cliListIssuesText(issues []Issue, total int, isSearch bool) {
	if len(issues) == 0 {
		fmt.Println("No issues matched")
		if isSearch {
			fmt.Println("Search: active")
		}
		fmt.Printf("Total: %d issues\n", total)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "SOURCE\tID\tSTATUS\tCREATED\tPROJECT\tREPORTER\tTITLE\n")
	fmt.Fprintf(w, "------\t--\t------\t-------\t-------\t--------\t-----\n")

	for _, i := range issues {
		status := "OPEN"
		if strings.EqualFold(i.Status, "resolved") {
			status = "DONE"
		}
		title := sanitizeTerminalText(i.Title)
		runes := []rune(title)
		if len(runes) > 68 {
			title = string(runes[:65]) + "…"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			sanitizeTerminalText(i.Source),
			sanitizeTerminalText(i.ShortID),
			status,
			FormatDate(i.FirstSeen),
			sanitizeTerminalText(i.Project.Slug),
			sanitizeTerminalText(i.Reporter),
			title,
		)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d issues\n", total)
}

func cliShowIssueText(issue Issue) {
	fmt.Printf("ID:          %s\n", sanitizeTerminalText(issue.ShortID))
	fmt.Printf("Source:      %s\n", sanitizeTerminalText(issue.Source))
	fmt.Printf("Project:     %s\n", sanitizeTerminalText(issue.Project.Slug))
	fmt.Printf("Title:       %s\n", sanitizeTerminalText(issue.Title))
	fmt.Printf("Status:      %s\n", sanitizeTerminalText(issue.Status))
	fmt.Printf("Reporter:    %s\n", sanitizeTerminalText(issue.Reporter))
	fmt.Printf("Events:      %s\n", sanitizeTerminalText(issue.Count))
	fmt.Printf("Users:       %d\n", issue.UserCount)
	fmt.Printf("Created:     %s\n", FormatDate(issue.FirstSeen))
	fmt.Printf("Last Seen:   %s\n", FormatDate(issue.LastSeen))
	if issue.URL != "" {
		fmt.Printf("URL:         %s\n", sanitizeTerminalText(issue.URL))
	}
	if issue.AssignedTo != nil {
		fmt.Printf("Assigned To: %s\n", sanitizeTerminalText(issue.AssignedTo.Name))
	}
}

func cliLeaderboardText(rows []cliLeaderboardRow, countEntries int, totalIssues int) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "RANK\tREPORTER\tCOUNT\tPERCENT\tBAR\n")
	fmt.Fprintf(w, "----\t--------\t-----\t-------\t---\n")
	for _, row := range rows {
		fmt.Fprintf(w, "%d\t%s\t%d\t%.1f%%\t%s\n", row.Rank, row.Reporter, row.Count, row.Percent, row.Bar)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d issues from %d reporters\n", totalIssues, countEntries)
}

func printCLICommandHelp(cmd string) {
	command, ok := cliCommandSchema[cmd]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printCLIHelp()
		return
	}
	fmt.Printf("Command: %s\nDescription: %s\nUsage: %s\n", command.Name, command.Description, command.Usage)
}

func printCLIHelp() {
	fmt.Print(`issuesherpa - IssueSherpa: unified Sentry + GitLab + GitHub issues

Usage:
  issuesherpa                              Launch TUI
  issuesherpa init                         Create a user config template
  issuesherpa list                         List all issues
  issuesherpa list --open                  List open issues only
  issuesherpa list --resolved              List resolved issues only
  issuesherpa search <query>               Search across title/project/reporter/id
  issuesherpa show <ISSUE-ID>              Show issue details
  issuesherpa leaderboard                  Show reporter leaderboard
  issuesherpa leaderboard --project <slug>  Leaderboard for a project
  issuesherpa version                      Show build version
  issuesherpa --version                    Show build version
  issuesherpa json                         Compatibility alias for list JSON output

Machine interface:
  --schema, --describe                    Print machine-readable command metadata
  --output <json|ndjson|text>             Choose output format (default: json)
  --pretty                                 Pretty-print JSON output
  --fields <comma-separated>               Select JSON fields
  --limit <n> / --offset <n>              Pagination controls

Filters and sorting:
  --project <slug>                        Filter by project/repo slug
  --source <sentry|gitlab|github>         Filter by source
  --search <query>                        Filter/search issues
  --sort <created|updated|project|reporter|status|title|source|id>
                                          Sort order
  --open                                  Show only open issues
  --resolved                              Show only resolved issues
  --desc                                  Sort descending
  --offline                               Use cached data only (no API calls)

For CLI consumers, default output is JSON. Use --output text for human tables.
`)
}
