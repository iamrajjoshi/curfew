package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rajjoshi/curfew/internal/config"
)

var (
	buildOnce  sync.Once
	binaryPath string
	buildErr   error
)

func TestCLIEndToEndForceCheckAndHistory(t *testing.T) {
	t.Parallel()

	env := newCLIEnv(t)

	result := runCurfew(t, env, "", "start")
	if result.exitCode != 0 {
		t.Fatalf("start exit code = %d, stderr = %s", result.exitCode, result.stderr)
	}
	if !strings.Contains(result.stdout, "Curfew force-enabled") {
		t.Fatalf("unexpected start stdout:\n%s", result.stdout)
	}

	blocked := runCurfew(t, env, "nope\n", "check", "--shell", "zsh", "--", "claude")
	if blocked.exitCode != 1 {
		t.Fatalf("blocked check exit code = %d, want 1\nstderr:\n%s", blocked.exitCode, blocked.stderr)
	}
	if !strings.Contains(blocked.stderr, "Type the passphrase to continue") {
		t.Fatalf("expected a passphrase prompt, got:\n%s", blocked.stderr)
	}

	allowed := runCurfew(t, env, "i am choosing to break my own rule\n", "check", "--shell", "zsh", "--", "claude")
	if allowed.exitCode != 0 {
		t.Fatalf("allowed check exit code = %d, stderr = %s", allowed.exitCode, allowed.stderr)
	}

	history := runCurfew(t, env, "", "history", "--days", "7")
	if history.exitCode != 0 {
		t.Fatalf("history exit code = %d, stderr = %s", history.exitCode, history.stderr)
	}
	if !strings.Contains(history.stdout, "overrode") {
		t.Fatalf("expected history to show an override, got:\n%s", history.stdout)
	}

	stats := runCurfew(t, env, "", "stats", "--days", "7")
	if stats.exitCode != 0 {
		t.Fatalf("stats exit code = %d, stderr = %s", stats.exitCode, stats.stderr)
	}
	if !strings.Contains(stats.stdout, "Most-attempted after-hours command: claude") {
		t.Fatalf("expected stats to mention claude, got:\n%s", stats.stdout)
	}
}

func TestCLIEndToEndInstallRulesAndStatusFallback(t *testing.T) {
	t.Parallel()

	env := newCLIEnv(t)
	home := env["HOME"]

	install := runCurfew(t, env, "", "install", "--shell", "zsh")
	if install.exitCode != 0 {
		t.Fatalf("install exit code = %d, stderr = %s", install.exitCode, install.stderr)
	}
	rcPath := filepath.Join(home, ".zshrc")
	bytes, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc file: %v", err)
	}
	if !strings.Contains(string(bytes), "curfew init zsh") {
		t.Fatalf("expected rc file to include the curfew init line, got:\n%s", string(bytes))
	}

	doctor := runCurfew(t, env, "", "doctor")
	if doctor.exitCode != 0 {
		t.Fatalf("doctor exit code = %d, stderr = %s", doctor.exitCode, doctor.stderr)
	}
	if !strings.Contains(doctor.stdout, "Managed block installed: true") {
		t.Fatalf("expected doctor to report the managed block, got:\n%s", doctor.stdout)
	}

	add := runCurfew(t, env, "", "rules", "add", "terraform plan*", "--action", "warn")
	if add.exitCode != 0 {
		t.Fatalf("rules add exit code = %d, stderr = %s", add.exitCode, add.stderr)
	}
	list := runCurfew(t, env, "", "rules")
	if list.exitCode != 0 {
		t.Fatalf("rules list exit code = %d, stderr = %s", list.exitCode, list.stderr)
	}
	if !strings.Contains(list.stdout, "terraform plan*") {
		t.Fatalf("expected rules output to include the added rule, got:\n%s", list.stdout)
	}

	remove := runCurfew(t, env, "", "rules", "rm", "terraform plan*")
	if remove.exitCode != 0 {
		t.Fatalf("rules rm exit code = %d, stderr = %s", remove.exitCode, remove.stderr)
	}

	status := runCurfew(t, env, "", "status")
	if status.exitCode != 0 {
		t.Fatalf("status exit code = %d, stderr = %s", status.exitCode, status.stderr)
	}
	root := runCurfew(t, env, "")
	if root.exitCode != 0 {
		t.Fatalf("root exit code = %d, stderr = %s", root.exitCode, root.stderr)
	}
	if root.stdout != status.stdout {
		t.Fatalf("expected root command to fall back to status output in non-interactive mode\nroot:\n%s\nstatus:\n%s", root.stdout, status.stdout)
	}

	initScript := runCurfew(t, env, "", "init", "bash")
	if initScript.exitCode != 0 {
		t.Fatalf("init bash exit code = %d, stderr = %s", initScript.exitCode, initScript.stderr)
	}
	if !strings.Contains(initScript.stdout, "bind -x") {
		t.Fatalf("expected bash init output to define readline bindings, got:\n%s", initScript.stdout)
	}
}

func TestCLIEndToEndSnoozeFlow(t *testing.T) {
	t.Parallel()

	env := newCLIEnv(t)

	start := runCurfew(t, env, "", "start")
	if start.exitCode != 0 {
		t.Fatalf("start exit code = %d, stderr = %s", start.exitCode, start.stderr)
	}

	snooze := runCurfew(t, env, "", "snooze", "2m")
	if snooze.exitCode != 0 {
		t.Fatalf("snooze exit code = %d, stderr = %s", snooze.exitCode, snooze.stderr)
	}
	if !strings.Contains(snooze.stdout, "Curfew snoozed") {
		t.Fatalf("expected snooze output, got:\n%s", snooze.stdout)
	}

	status := runCurfew(t, env, "", "status")
	if status.exitCode != 0 {
		t.Fatalf("status exit code = %d, stderr = %s", status.exitCode, status.stderr)
	}
	if !strings.Contains(status.stdout, "Curfew snoozed until") {
		t.Fatalf("expected status to show a snooze, got:\n%s", status.stdout)
	}
}

func newCLIEnv(t *testing.T) map[string]string {
	t.Helper()

	dir := t.TempDir()
	cfg := config.Default()
	cfg.Schedule.Timezone = "America/Los_Angeles"

	xdgConfig := filepath.Join(dir, ".config")
	xdgData := filepath.Join(dir, ".local", "share")
	xdgState := filepath.Join(dir, ".local", "state")
	if err := os.MkdirAll(filepath.Join(xdgConfig, "curfew"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := config.Save(filepath.Join(xdgConfig, "curfew", "config.toml"), cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	return map[string]string{
		"HOME":            dir,
		"XDG_CONFIG_HOME": xdgConfig,
		"XDG_DATA_HOME":   xdgData,
		"XDG_STATE_HOME":  xdgState,
		"SHELL":           "/bin/zsh",
	}
}

type cliResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func runCurfew(t *testing.T, env map[string]string, stdin string, args ...string) cliResult {
	t.Helper()

	command := exec.Command(mustBuildBinary(t), args...)
	command.Dir = repoRoot(t)

	environment := os.Environ()
	for key, value := range env {
		environment = append(environment, key+"="+value)
	}
	command.Env = environment
	command.Stdin = strings.NewReader(stdin)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	result := cliResult{
		stdout: stdout.String(),
		stderr: stderr.String(),
	}
	if err == nil {
		return result
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("run curfew %v: %v", args, err)
	}
	result.exitCode = exitErr.ExitCode()
	return result
}

func mustBuildBinary(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		tempDir, err := os.MkdirTemp("", "curfew-e2e-*")
		if err != nil {
			buildErr = err
			return
		}
		binaryPath = filepath.Join(tempDir, "curfew")
		command := exec.Command("go", "build", "-o", binaryPath, ".")
		command.Dir = repoRoot(t)
		output, err := command.CombinedOutput()
		if err != nil {
			buildErr = &buildFailure{err: err, output: string(output)}
		}
	})

	if buildErr != nil {
		t.Fatal(buildErr)
	}
	return binaryPath
}

func repoRoot(t *testing.T) string {
	t.Helper()

	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return root
}

type buildFailure struct {
	err    error
	output string
}

func (b *buildFailure) Error() string {
	return b.err.Error() + "\n" + b.output
}
