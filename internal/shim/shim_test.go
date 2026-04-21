package shim

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iamrajjoshi/curfew/internal/config"
	"github.com/iamrajjoshi/curfew/internal/paths"
)

func TestCommandsDeriveFromCriticalSetAndRules(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Rules: config.RulesConfig{
			Rule: []config.RuleEntry{
				{Pattern: "git push*", Action: "warn"},
				{Pattern: "terraform apply*", Action: "block"},
				{Pattern: "aider*", Action: "delay"},
				{Pattern: "cd repo", Action: "block"},
				{Pattern: "./deploy", Action: "block"},
				{Pattern: "claude", Action: "allow"},
			},
		},
	}

	got := Commands(cfg)
	want := []string{"aider", "claude", "claude-code", "cursor-agent", "git", "terraform"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("commands = %v, want %v", got, want)
	}
}

func TestWrapperIncludesCheckAndBypassPaths(t *testing.T) {
	t.Parallel()

	script := Wrapper("claude", "/tmp/curfew/shims")
	want := []string{
		"CURFEW_COMMAND='claude'",
		"CURFEW_SHIM_DIR='/tmp/curfew/shims'",
		"curfew_resolve_real()",
		"if [ -n \"${" + envHookActive + ":-}\" ] || [ -n \"${" + envShimActive + ":-}\" ]; then",
		envShimActive + "=1 command curfew check -- \"$CURFEW_COMMAND\" \"$@\"",
		"printf 'curfew shim warning: curfew check failed for %s (exit %s); running real binary\\n'",
		"exit 127",
	}
	for _, fragment := range want {
		if !strings.Contains(script, fragment) {
			t.Fatalf("wrapper missing %q:\n%s", fragment, script)
		}
	}
}

func TestInstallAndDiagnosePathBlockAndWrappers(t *testing.T) {
	t.Parallel()

	layout := testLayout(t)
	cfg := config.Config{
		Rules: config.RulesConfig{
			Rule: []config.RuleEntry{{Pattern: "terraform apply*", Action: "block"}},
		},
	}

	result, err := Install(layout, "zsh", cfg)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !result.PathChanged {
		t.Fatal("expected first install to change the rc file")
	}
	if len(result.Installed) == 0 {
		t.Fatal("expected wrappers to be installed")
	}

	diagnostics, err := Diagnose(layout, "zsh", cfg)
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	if !diagnostics.PathBlockInstalled {
		t.Fatal("expected path block to be installed")
	}
	if !contains(diagnostics.InstalledCommands, "terraform") {
		t.Fatalf("expected installed wrappers to include terraform, got %v", diagnostics.InstalledCommands)
	}

	if err := os.WriteFile(filepath.Join(layout.ShimDir(), "extra-tool"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write extra wrapper: %v", err)
	}
	if err := os.Remove(filepath.Join(layout.ShimDir(), "terraform")); err != nil {
		t.Fatalf("remove terraform wrapper: %v", err)
	}
	diagnostics, err = Diagnose(layout, "zsh", cfg)
	if err != nil {
		t.Fatalf("diagnose after drift: %v", err)
	}
	if !contains(diagnostics.MissingCommands, "terraform") {
		t.Fatalf("expected terraform to be missing, got %v", diagnostics.MissingCommands)
	}
	if !contains(diagnostics.ExtraCommands, "extra-tool") {
		t.Fatalf("expected extra-tool to be extra, got %v", diagnostics.ExtraCommands)
	}

	uninstall, err := Uninstall(layout, "zsh")
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if !uninstall.PathChanged {
		t.Fatal("expected uninstall to remove the path block")
	}
	if uninstall.Removed == 0 {
		t.Fatal("expected uninstall to remove shim wrappers")
	}
	if _, err := os.Stat(layout.ShimDir()); !os.IsNotExist(err) {
		t.Fatalf("expected shim dir to be removed, got err=%v", err)
	}
}

func TestPathBlocksAreShellSpecific(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind string
		want string
	}{
		{kind: "zsh", want: `export PATH='/tmp/curfew/shims':"$PATH"`},
		{kind: "bash", want: `export PATH='/tmp/curfew/shims':"$PATH"`},
		{kind: "fish", want: `contains -- '/tmp/curfew/shims' $PATH; or set -gx PATH '/tmp/curfew/shims' $PATH`},
	}

	for _, test := range tests {
		test := test
		t.Run(test.kind, func(t *testing.T) {
			t.Parallel()

			block, err := PathBlock(test.kind, "/tmp/curfew/shims")
			if err != nil {
				t.Fatalf("path block: %v", err)
			}
			if !strings.Contains(block, test.want) {
				t.Fatalf("block for %s missing %q:\n%s", test.kind, test.want, block)
			}
		})
	}
}

func TestWrapperExecutionPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		curfewScript    string
		command         string
		args            []string
		extraEnv        map[string]string
		wantExit        int
		wantStdout      string
		wantStderr      string
		createRealBin   bool
		realBinContents string
	}{
		{
			name:            "allow executes real binary",
			curfewScript:    "#!/bin/sh\nexit 0\n",
			command:         "claude-code",
			args:            []string{"hi"},
			wantStdout:      "real claude-code hi\n",
			createRealBin:   true,
			realBinContents: "#!/bin/sh\nprintf 'real claude-code %s\\n' \"$1\"\n",
		},
		{
			name:            "block exits one without real binary",
			curfewScript:    "#!/bin/sh\nexit 1\n",
			command:         "claude",
			wantExit:        1,
			createRealBin:   true,
			realBinContents: "#!/bin/sh\necho should-not-run\n",
		},
		{
			name:            "hook active bypasses curfew",
			curfewScript:    "#!/bin/sh\necho should-not-run >&2\nexit 99\n",
			command:         "claude",
			extraEnv:        map[string]string{envHookActive: "1"},
			wantStdout:      "hook bypass\n",
			createRealBin:   true,
			realBinContents: "#!/bin/sh\necho hook bypass\n",
		},
		{
			name:            "unexpected curfew failure fails open",
			curfewScript:    "#!/bin/sh\nexit 42\n",
			command:         "claude",
			wantStdout:      "fail open\n",
			wantStderr:      "curfew shim warning:",
			createRealBin:   true,
			realBinContents: "#!/bin/sh\necho fail open\n",
		},
		{
			name:            "missing curfew command fails open",
			command:         "claude",
			wantStdout:      "missing curfew fallback\n",
			wantStderr:      "curfew shim warning:",
			createRealBin:   true,
			realBinContents: "#!/bin/sh\necho missing curfew fallback\n",
		},
		{
			name:       "missing real binary exits 127",
			command:    "claude",
			wantExit:   127,
			wantStderr: "could not find real claude",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			layout := testLayout(t)
			if err := os.MkdirAll(layout.ShimDir(), 0o755); err != nil {
				t.Fatalf("mkdir shim dir: %v", err)
			}
			wrapperPath := filepath.Join(layout.ShimDir(), test.command)
			if err := os.WriteFile(wrapperPath, []byte(Wrapper(test.command, layout.ShimDir())), 0o755); err != nil {
				t.Fatalf("write wrapper: %v", err)
			}

			realBinDir := filepath.Join(t.TempDir(), "real")
			if err := os.MkdirAll(realBinDir, 0o755); err != nil {
				t.Fatalf("mkdir real bin dir: %v", err)
			}
			if test.createRealBin {
				if err := os.WriteFile(filepath.Join(realBinDir, test.command), []byte(test.realBinContents), 0o755); err != nil {
					t.Fatalf("write real binary: %v", err)
				}
			}

			curfewBinDir := filepath.Join(t.TempDir(), "bin")
			if err := os.MkdirAll(curfewBinDir, 0o755); err != nil {
				t.Fatalf("mkdir curfew bin dir: %v", err)
			}
			if test.curfewScript != "" {
				if err := os.WriteFile(filepath.Join(curfewBinDir, "curfew"), []byte(test.curfewScript), 0o755); err != nil {
					t.Fatalf("write fake curfew: %v", err)
				}
			}

			command := exec.Command(wrapperPath, test.args...)
			env := []string{"PATH=" + strings.Join([]string{layout.ShimDir(), realBinDir, curfewBinDir}, ":")}
			for key, value := range test.extraEnv {
				env = append(env, key+"="+value)
			}
			command.Env = env
			output, err := command.CombinedOutput()
			if test.wantExit == 0 && err != nil {
				t.Fatalf("wrapper failed: %v\n%s", err, output)
			}
			if test.wantExit != 0 {
				if err == nil {
					t.Fatalf("wrapper exit code = 0, want %d\n%s", test.wantExit, output)
				}
				exitErr, ok := err.(*exec.ExitError)
				if !ok {
					t.Fatalf("wrapper error: %v", err)
				}
				if exitErr.ExitCode() != test.wantExit {
					t.Fatalf("wrapper exit code = %d, want %d\n%s", exitErr.ExitCode(), test.wantExit, output)
				}
			}
			if test.wantStdout != "" && !strings.Contains(string(output), test.wantStdout) {
				t.Fatalf("expected output to contain %q, got:\n%s", test.wantStdout, output)
			}
			if test.wantStderr != "" && !strings.Contains(string(output), test.wantStderr) {
				t.Fatalf("expected output to contain %q, got:\n%s", test.wantStderr, output)
			}
		})
	}
}

func testLayout(t *testing.T) paths.Layout {
	t.Helper()

	dir := t.TempDir()
	return paths.Layout{
		Home:       dir,
		ConfigHome: filepath.Join(dir, ".config"),
		DataHome:   filepath.Join(dir, ".local", "share"),
		StateHome:  filepath.Join(dir, ".local", "state"),
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
