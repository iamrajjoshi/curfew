package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/iamrajjoshi/curfew/internal/config"
	"github.com/iamrajjoshi/curfew/internal/paths"
	"github.com/iamrajjoshi/curfew/internal/shell"
)

const (
	beginMarker = "# >>> curfew shim path >>>"
	endMarker   = "# <<< curfew shim path <<<"

	envHookActive = "CURFEW_HOOK_ACTIVE"
	envShimActive = "CURFEW_SHIM_ACTIVE"
)

var (
	criticalCommands = []string{"aider", "claude", "claude-code", "cursor-agent"}
	skipCommands     = map[string]struct{}{
		".":       {},
		"alias":   {},
		"bg":      {},
		"builtin": {},
		"cd":      {},
		"curfew":  {},
		"eval":    {},
		"exec":    {},
		"exit":    {},
		"export":  {},
		"fg":      {},
		"history": {},
		"jobs":    {},
		"pwd":     {},
		"read":    {},
		"set":     {},
		"source":  {},
		"unset":   {},
	}
)

type Diagnostics struct {
	RCPath             string
	ShimDir            string
	PathBlockInstalled bool
	InstalledCommands  []string
	ExpectedCommands   []string
	MissingCommands    []string
	ExtraCommands      []string
}

type InstallResult struct {
	RCPath        string
	ShimDir       string
	PathChanged   bool
	Installed     []string
	ExpectedCount int
}

type UninstallResult struct {
	RCPath      string
	ShimDir     string
	PathChanged bool
	Removed     int
}

func Commands(cfg config.Config) []string {
	set := make(map[string]struct{}, len(criticalCommands)+len(cfg.Rules.Rule))
	for _, command := range criticalCommands {
		set[command] = struct{}{}
	}
	for _, rule := range cfg.Rules.Rule {
		if strings.EqualFold(strings.TrimSpace(rule.Action), "allow") {
			continue
		}
		command := normalizeCommandWord(rule.Pattern)
		if command == "" {
			continue
		}
		if _, skip := skipCommands[command]; skip {
			continue
		}
		set[command] = struct{}{}
	}
	commands := make([]string, 0, len(set))
	for command := range set {
		commands = append(commands, command)
	}
	slices.Sort(commands)
	return commands
}

func Diagnose(layout paths.Layout, kind string, cfg config.Config) (Diagnostics, error) {
	rcPath, installed, err := pathBlockInstalled(layout, kind)
	if err != nil {
		return Diagnostics{}, err
	}
	expected := Commands(cfg)
	current, err := installedCommands(layout)
	if err != nil {
		return Diagnostics{}, err
	}
	missing, extra := diffCommands(expected, current)
	return Diagnostics{
		RCPath:             rcPath,
		ShimDir:            layout.ShimDir(),
		PathBlockInstalled: installed,
		InstalledCommands:  current,
		ExpectedCommands:   expected,
		MissingCommands:    missing,
		ExtraCommands:      extra,
	}, nil
}

func Install(layout paths.Layout, kind string, cfg config.Config) (InstallResult, error) {
	commands := Commands(cfg)
	if err := writeWrappers(layout, commands); err != nil {
		return InstallResult{}, err
	}
	rcPath, changed, err := installPathBlock(layout, kind)
	if err != nil {
		return InstallResult{}, err
	}
	return InstallResult{
		RCPath:        rcPath,
		ShimDir:       layout.ShimDir(),
		PathChanged:   changed,
		Installed:     commands,
		ExpectedCount: len(commands),
	}, nil
}

func Uninstall(layout paths.Layout, kind string) (UninstallResult, error) {
	current, err := installedCommands(layout)
	if err != nil {
		return UninstallResult{}, err
	}
	rcPath, changed, err := uninstallPathBlock(layout, kind)
	if err != nil {
		return UninstallResult{}, err
	}
	if err := os.RemoveAll(layout.ShimDir()); err != nil {
		return UninstallResult{}, err
	}
	return UninstallResult{
		RCPath:      rcPath,
		ShimDir:     layout.ShimDir(),
		PathChanged: changed,
		Removed:     len(current),
	}, nil
}

func PathBlock(kind string, shimDir string) (string, error) {
	quotedShimDir := shellQuote(shimDir)
	switch kind {
	case "zsh", "bash":
		return strings.Join([]string{
			beginMarker,
			`case ":$PATH:" in`,
			`  *:` + quotedShimDir + `:*) ;;`,
			`  *) export PATH=` + quotedShimDir + `:"$PATH" ;;`,
			`esac`,
			endMarker,
			"",
		}, "\n"), nil
	case "fish":
		return strings.Join([]string{
			beginMarker,
			"contains -- " + quotedShimDir + " $PATH; or set -gx PATH " + quotedShimDir + " $PATH",
			endMarker,
			"",
		}, "\n"), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", kind)
	}
}

func Wrapper(command string, shimDir string) string {
	quotedCommand := shellQuote(command)
	quotedShimDir := shellQuote(shimDir)
	return strings.Join([]string{
		"#!/bin/sh",
		"CURFEW_COMMAND=" + quotedCommand,
		"CURFEW_SHIM_DIR=" + quotedShimDir,
		"",
		"curfew_resolve_real() {",
		"  old_ifs=$IFS",
		"  IFS=:",
		"  for entry in $PATH; do",
		"    [ -n \"$entry\" ] || entry=.",
		"    if [ \"$entry\" = \"$CURFEW_SHIM_DIR\" ]; then",
		"      continue",
		"    fi",
		"    candidate=\"$entry/$CURFEW_COMMAND\"",
		"    if [ -x \"$candidate\" ] && [ ! -d \"$candidate\" ]; then",
		"      IFS=$old_ifs",
		"      printf '%s\\n' \"$candidate\"",
		"      return 0",
		"    fi",
		"  done",
		"  IFS=$old_ifs",
		"  return 1",
		"}",
		"",
		"real_bin=$(curfew_resolve_real)",
		"if [ -z \"$real_bin\" ]; then",
		"  printf 'curfew shim: could not find real %s in PATH\\n' \"$CURFEW_COMMAND\" >&2",
		"  exit 127",
		"fi",
		"",
		"if [ -n \"${" + envHookActive + ":-}\" ] || [ -n \"${" + envShimActive + ":-}\" ]; then",
		"  exec \"$real_bin\" \"$@\"",
		"fi",
		"",
		envShimActive + "=1 command curfew check -- \"$CURFEW_COMMAND\" \"$@\"",
		"status=$?",
		"if [ \"$status\" -eq 0 ]; then",
		"  exec \"$real_bin\" \"$@\"",
		"fi",
		"if [ \"$status\" -eq 1 ]; then",
		"  exit 1",
		"fi",
		"",
		"printf 'curfew shim warning: curfew check failed for %s (exit %s); running real binary\\n' \"$CURFEW_COMMAND\" \"$status\" >&2",
		"exec \"$real_bin\" \"$@\"",
		"",
	}, "\n")
}

func normalizeCommandWord(pattern string) string {
	fields := strings.Fields(strings.TrimSpace(pattern))
	if len(fields) == 0 {
		return ""
	}
	command := strings.TrimSuffix(strings.ToLower(fields[0]), "*")
	if command == "" || strings.Contains(command, "/") {
		return ""
	}
	return command
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func pathBlockInstalled(layout paths.Layout, kind string) (string, bool, error) {
	rcPath, err := shell.RCPath(layout, kind)
	if err != nil {
		return "", false, err
	}
	bytes, err := os.ReadFile(rcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return rcPath, false, nil
		}
		return "", false, err
	}
	text := string(bytes)
	return rcPath, strings.Contains(text, beginMarker) && strings.Contains(text, endMarker), nil
}

func installPathBlock(layout paths.Layout, kind string) (string, bool, error) {
	rcPath, err := shell.RCPath(layout, kind)
	if err != nil {
		return "", false, err
	}
	block, err := PathBlock(kind, layout.ShimDir())
	if err != nil {
		return "", false, err
	}
	if err := os.MkdirAll(filepath.Dir(rcPath), 0o755); err != nil {
		return "", false, err
	}
	existing, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return "", false, err
	}
	text := string(existing)
	if strings.Contains(text, beginMarker) && strings.Contains(text, endMarker) {
		return rcPath, false, nil
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += block
	if err := os.WriteFile(rcPath, []byte(text), 0o644); err != nil {
		return "", false, err
	}
	return rcPath, true, nil
}

func uninstallPathBlock(layout paths.Layout, kind string) (string, bool, error) {
	rcPath, err := shell.RCPath(layout, kind)
	if err != nil {
		return "", false, err
	}
	bytes, err := os.ReadFile(rcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return rcPath, false, nil
		}
		return "", false, err
	}
	text := string(bytes)
	start := strings.Index(text, beginMarker)
	end := strings.Index(text, endMarker)
	if start == -1 || end == -1 {
		return rcPath, false, nil
	}
	end += len(endMarker)
	updated := strings.TrimLeft(text[:start]+text[end:], "\n")
	if updated != "" && !strings.HasSuffix(updated, "\n") {
		updated += "\n"
	}
	if err := os.WriteFile(rcPath, []byte(updated), 0o644); err != nil {
		return "", false, err
	}
	return rcPath, true, nil
}

func writeWrappers(layout paths.Layout, commands []string) error {
	if err := os.RemoveAll(layout.ShimDir()); err != nil {
		return err
	}
	if err := os.MkdirAll(layout.ShimDir(), 0o755); err != nil {
		return err
	}
	for _, command := range commands {
		path := filepath.Join(layout.ShimDir(), command)
		if err := os.WriteFile(path, []byte(Wrapper(command, layout.ShimDir())), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func installedCommands(layout paths.Layout) ([]string, error) {
	entries, err := os.ReadDir(layout.ShimDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	commands := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		commands = append(commands, entry.Name())
	}
	slices.Sort(commands)
	return commands, nil
}

func diffCommands(expected []string, actual []string) ([]string, []string) {
	expectedSet := make(map[string]struct{}, len(expected))
	for _, command := range expected {
		expectedSet[command] = struct{}{}
	}
	actualSet := make(map[string]struct{}, len(actual))
	for _, command := range actual {
		actualSet[command] = struct{}{}
	}

	var missing []string
	for _, command := range expected {
		if _, ok := actualSet[command]; !ok {
			missing = append(missing, command)
		}
	}

	var extra []string
	for _, command := range actual {
		if _, ok := expectedSet[command]; !ok {
			extra = append(extra, command)
		}
	}
	return missing, extra
}
