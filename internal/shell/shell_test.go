package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iamrajjoshi/curfew/internal/paths"
)

func TestInstallAndUninstallManagedBlock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	layout := paths.Layout{
		Home:       dir,
		ConfigHome: filepath.Join(dir, ".config"),
		DataHome:   filepath.Join(dir, ".local", "share"),
		StateHome:  filepath.Join(dir, ".local", "state"),
	}

	rcPath, changed, err := Install(layout, "zsh")
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !changed {
		t.Fatal("expected first install to change the rc file")
	}
	content, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, beginMarker) || !strings.Contains(text, endMarker) {
		t.Fatalf("managed block missing from rc file:\n%s", text)
	}

	_, changed, err = Install(layout, "zsh")
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if changed {
		t.Fatal("expected second install to be idempotent")
	}

	_, changed, err = Uninstall(layout, "zsh")
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if !changed {
		t.Fatal("expected uninstall to remove the managed block")
	}
	content, err = os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc after uninstall: %v", err)
	}
	if strings.Contains(string(content), beginMarker) {
		t.Fatalf("managed block still present after uninstall:\n%s", string(content))
	}
}

func TestDetectAndRCPath(t *testing.T) {
	t.Parallel()

	layout := paths.Layout{
		Home:       "/tmp/curfew-home",
		ConfigHome: "/tmp/curfew-home/.config",
	}
	tests := []struct {
		name       string
		explicit   string
		shellEnv   string
		wantKind   string
		wantSuffix string
	}{
		{name: "explicit shell wins", explicit: "bash", shellEnv: "/bin/zsh", wantKind: "bash", wantSuffix: ".bashrc"},
		{name: "detect zsh from env", shellEnv: "/bin/zsh", wantKind: "zsh", wantSuffix: ".zshrc"},
		{name: "detect fish from env", shellEnv: "/opt/homebrew/bin/fish", wantKind: "fish", wantSuffix: filepath.Join(".config", "fish", "config.fish")},
		{name: "default shell", shellEnv: "", wantKind: "zsh", wantSuffix: ".zshrc"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			kind := Detect(test.explicit, test.shellEnv)
			if kind != test.wantKind {
				t.Fatalf("detected shell = %q, want %q", kind, test.wantKind)
			}
			path, err := RCPath(layout, kind)
			if err != nil {
				t.Fatalf("rc path: %v", err)
			}
			if !strings.HasSuffix(path, test.wantSuffix) {
				t.Fatalf("rc path = %q, want suffix %q", path, test.wantSuffix)
			}
		})
	}
}

func TestInitScriptsReferenceCurfewCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind string
		want []string
	}{
		{
			kind: "zsh",
			want: []string{
				"[[ -o interactive ]] || return 0",
				"[[ -n ${" + envInitDone + ":-} ]] && return 0",
				"export " + envShellHook + "=1",
				"export " + envShellKind + "=zsh",
				"${" + envHookActive + ":-} || \"$cmd\" == curfew || \"$cmd\" == curfew\\ *",
				envHookActive + "=1 command curfew check --shell zsh -- \"$cmd\"",
				"zle .accept-line",
				"zle redisplay",
			},
		},
		{
			kind: "bash",
			want: []string{
				"[[ $- == *i* ]] || return 0",
				"[[ -n ${" + envInitDone + ":-} ]] && return 0",
				"export " + envShellHook + "=1",
				"export " + envShellKind + "=bash",
				"${" + envHookActive + ":-} || \"$cmd\" == curfew || \"$cmd\" == curfew\\ *",
				envHookActive + "=1 command curfew check --shell bash -- \"$cmd\"",
				"eval -- \"$cmd\"",
				"READLINE_POINT=${#READLINE_LINE}",
			},
		},
		{
			kind: "fish",
			want: []string{
				"status is-interactive; or return",
				"set -q " + envInitDone + "; and return",
				"set -gx " + envShellHook + " 1",
				"set -gx " + envShellKind + " fish",
				"if set -q " + envHookActive + "; or string match -rq '^\\s*curfew(\\s|$)' -- $cmd",
				"command curfew check --shell fish -- \"$cmd\"",
				"commandline -f execute",
				"commandline -f repaint",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.kind, func(t *testing.T) {
			t.Parallel()

			script, err := Init(test.kind)
			if err != nil {
				t.Fatalf("init: %v", err)
			}
			for _, want := range test.want {
				if !strings.Contains(script, want) {
					t.Fatalf("script for %s missing %q:\n%s", test.kind, want, script)
				}
			}
		})
	}
}

func TestManagedBlocksUseCanonicalInitCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind string
		want string
	}{
		{kind: "zsh", want: `eval "$(curfew init zsh)"`},
		{kind: "bash", want: `eval "$(curfew init bash)"`},
		{kind: "fish", want: "curfew init fish | source"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.kind, func(t *testing.T) {
			t.Parallel()

			block, err := ManagedBlock(test.kind)
			if err != nil {
				t.Fatalf("managed block: %v", err)
			}
			if !strings.Contains(block, test.want) {
				t.Fatalf("managed block for %s missing %q:\n%s", test.kind, test.want, block)
			}
		})
	}
}

func TestDiagnoseReportsShellAndHookState(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	layout := paths.Layout{
		Home:       dir,
		ConfigHome: filepath.Join(dir, ".config"),
		DataHome:   filepath.Join(dir, ".local", "share"),
		StateHome:  filepath.Join(dir, ".local", "state"),
	}
	if _, _, err := Install(layout, "fish"); err != nil {
		t.Fatalf("install fish: %v", err)
	}

	diagnostics, err := Diagnose(layout, "", "/usr/local/bin/fish", "fish", true)
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	if diagnostics.DetectedShell != "fish" {
		t.Fatalf("detected shell = %q, want fish", diagnostics.DetectedShell)
	}
	if !diagnostics.ManagedBlockInstalled {
		t.Fatal("expected managed block to be installed")
	}
	if !strings.HasSuffix(diagnostics.RCPath, "/.config/fish/config.fish") {
		t.Fatalf("unexpected rc path: %s", diagnostics.RCPath)
	}
	if !diagnostics.HookActive {
		t.Fatal("expected hook to be active")
	}
	if diagnostics.HookShell != "fish" {
		t.Fatalf("hook shell = %q, want fish", diagnostics.HookShell)
	}
}
