package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rajjoshi/curfew/internal/paths"
)

const (
	beginMarker = "# >>> curfew initialize >>>"
	endMarker   = "# <<< curfew initialize <<<"
)

func Detect(explicit string, shellEnv string) string {
	if explicit != "" {
		return strings.ToLower(strings.TrimSpace(explicit))
	}
	base := filepath.Base(strings.TrimSpace(shellEnv))
	if base == "" {
		return "zsh"
	}
	return strings.ToLower(base)
}

func Supported(kind string) bool {
	switch kind {
	case "zsh", "bash", "fish":
		return true
	default:
		return false
	}
}

func RCPath(layout paths.Layout, kind string) (string, error) {
	switch kind {
	case "zsh":
		return filepath.Join(layout.Home, ".zshrc"), nil
	case "bash":
		return filepath.Join(layout.Home, ".bashrc"), nil
	case "fish":
		return filepath.Join(layout.ConfigHome, "fish", "config.fish"), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", kind)
	}
}

func ManagedBlock(kind string) (string, error) {
	switch kind {
	case "zsh", "bash":
		return strings.Join([]string{
			beginMarker,
			fmt.Sprintf("if command -v curfew >/dev/null 2>&1; then"),
			fmt.Sprintf("  eval \"$(curfew init %s)\"", kind),
			"fi",
			endMarker,
			"",
		}, "\n"), nil
	case "fish":
		return strings.Join([]string{
			beginMarker,
			"if status is-interactive",
			"  command -sq curfew; and curfew init fish | source",
			"end",
			endMarker,
			"",
		}, "\n"), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", kind)
	}
}

func Install(layout paths.Layout, kind string) (string, bool, error) {
	rcPath, err := RCPath(layout, kind)
	if err != nil {
		return "", false, err
	}
	block, err := ManagedBlock(kind)
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

func Uninstall(layout paths.Layout, kind string) (string, bool, error) {
	rcPath, err := RCPath(layout, kind)
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

func Installed(layout paths.Layout, kind string) (string, bool, error) {
	rcPath, err := RCPath(layout, kind)
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

func Init(kind string) (string, error) {
	switch kind {
	case "zsh":
		return zshInit(), nil
	case "bash":
		return bashInit(), nil
	case "fish":
		return fishInit(), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", kind)
	}
}

func zshInit() string {
	return strings.Join([]string{
		"[[ -o interactive ]] || return 0",
		"[[ -n ${CURFEW_INIT_DONE:-} ]] && return 0",
		"export CURFEW_INIT_DONE=1",
		"export CURFEW_SHELL_HOOK=1",
		"export CURFEW_SHELL_KIND=zsh",
		"function __curfew_accept_line() {",
		"  emulate -L zsh",
		"  local cmd=\"$BUFFER\"",
		"  if [[ -z ${cmd//[[:space:]]/} ]]; then",
		"    zle .accept-line",
		"    return",
		"  fi",
		"  if [[ -n ${CURFEW_HOOK_ACTIVE:-} || \"$cmd\" == curfew || \"$cmd\" == curfew\\ * ]]; then",
		"    zle .accept-line",
		"    return",
		"  fi",
		"  CURFEW_HOOK_ACTIVE=1 command curfew check --shell zsh -- \"$cmd\"",
		"  local exit_status=$?",
		"  unset CURFEW_HOOK_ACTIVE",
		"  if (( exit_status == 0 )); then",
		"    zle .accept-line",
		"  else",
		"    zle redisplay",
		"  fi",
		"}",
		"zle -N accept-line __curfew_accept_line",
		"",
	}, "\n")
}

func bashInit() string {
	return strings.Join([]string{
		"[[ $- == *i* ]] || return 0",
		"[[ -n ${CURFEW_INIT_DONE:-} ]] && return 0",
		"export CURFEW_INIT_DONE=1",
		"export CURFEW_SHELL_HOOK=1",
		"export CURFEW_SHELL_KIND=bash",
		"__curfew_accept_line() {",
		"  local cmd=\"$READLINE_LINE\"",
		"  if [[ -z ${cmd//[[:space:]]/} ]]; then",
		"    printf '\\n'",
		"    READLINE_LINE=''",
		"    READLINE_POINT=0",
		"    return",
		"  fi",
		"  if [[ -n ${CURFEW_HOOK_ACTIVE:-} || \"$cmd\" == curfew || \"$cmd\" == curfew\\ * ]]; then",
		"    builtin history -s -- \"$cmd\"",
		"    printf '\\n'",
		"    READLINE_LINE=''",
		"    READLINE_POINT=0",
		"    eval -- \"$cmd\"",
		"    return",
		"  fi",
		"  CURFEW_HOOK_ACTIVE=1 command curfew check --shell bash -- \"$cmd\"",
		"  local status=$?",
		"  unset CURFEW_HOOK_ACTIVE",
		"  if [[ $status -eq 0 ]]; then",
		"    builtin history -s -- \"$cmd\"",
		"    printf '\\n'",
		"    READLINE_LINE=''",
		"    READLINE_POINT=0",
		"    eval -- \"$cmd\"",
		"  else",
		"    READLINE_POINT=${#READLINE_LINE}",
		"  fi",
		"}",
		"bind -x '\"\\C-m\":__curfew_accept_line'",
		"bind -x '\"\\C-j\":__curfew_accept_line'",
		"",
	}, "\n")
}

func fishInit() string {
	return strings.Join([]string{
		"status is-interactive; or return",
		"set -q CURFEW_INIT_DONE; and return",
		"set -gx CURFEW_INIT_DONE 1",
		"set -gx CURFEW_SHELL_HOOK 1",
		"set -gx CURFEW_SHELL_KIND fish",
		"function __curfew_execute",
		"  set -l cmd (commandline)",
		"  if test -z (string trim -- $cmd)",
		"    commandline -f execute",
		"    return",
		"  end",
		"  if set -q CURFEW_HOOK_ACTIVE; or string match -rq '^\\s*curfew(\\s|$)' -- $cmd",
		"    commandline -f execute",
		"    return",
		"  end",
		"  set -gx CURFEW_HOOK_ACTIVE 1",
		"  command curfew check --shell fish -- \"$cmd\"",
		"  set -l status $status",
		"  set -e CURFEW_HOOK_ACTIVE",
		"  if test $status -eq 0",
		"    commandline -f execute",
		"  else",
		"    commandline -f repaint",
		"  end",
		"end",
		"if functions -q fish_user_key_bindings",
		"  functions -c fish_user_key_bindings __curfew_orig_fish_user_key_bindings",
		"end",
		"function fish_user_key_bindings",
		"  if functions -q __curfew_orig_fish_user_key_bindings",
		"    __curfew_orig_fish_user_key_bindings",
		"  end",
		"  for keymap in default insert",
		"    bind -M $keymap \\r __curfew_execute",
		"    bind -M $keymap \\n __curfew_execute",
		"  end",
		"end",
		"fish_user_key_bindings",
		"",
	}, "\n")
}
