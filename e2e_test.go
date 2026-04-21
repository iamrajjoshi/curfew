package main_test

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
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

func TestCLIVersionSurface(t *testing.T) {
	t.Parallel()

	env := newCLIEnv(t)

	command := runCurfew(t, env, "", "version")
	if command.exitCode != 0 {
		t.Fatalf("version exit code = %d, stderr = %s", command.exitCode, command.stderr)
	}
	if strings.TrimSpace(command.stdout) != "curfew dev" {
		t.Fatalf("version stdout = %q, want %q", command.stdout, "curfew dev\n")
	}

	flag := runCurfew(t, env, "", "--version")
	if flag.exitCode != 0 {
		t.Fatalf("--version exit code = %d, stderr = %s", flag.exitCode, flag.stderr)
	}
	if strings.TrimSpace(flag.stdout) != "curfew dev" {
		t.Fatalf("--version stdout = %q, want %q", flag.stdout, "curfew dev\n")
	}

	help := runCurfew(t, env, "", "version", "--help")
	if help.exitCode != 0 {
		t.Fatalf("version --help exit code = %d, stderr = %s", help.exitCode, help.stderr)
	}
	if !strings.Contains(help.stdout, "Show curfew version") {
		t.Fatalf("expected version help output, got:\n%s", help.stdout)
	}

	extra := runCurfew(t, env, "", "version", "extra")
	if extra.exitCode == 0 {
		t.Fatalf("version extra should fail, stdout = %s", extra.stdout)
	}
	if !strings.Contains(extra.stderr, "unknown command \"extra\" for \"curfew version\"") {
		t.Fatalf("unexpected version extra stderr:\n%s", extra.stderr)
	}

	flagExtra := runCurfew(t, env, "", "--version", "extra")
	if flagExtra.exitCode == 0 {
		t.Fatalf("--version extra should fail, stdout = %s", flagExtra.stdout)
	}
	if !strings.Contains(flagExtra.stderr, "unknown command \"extra\" for \"curfew\"") {
		t.Fatalf("unexpected --version extra stderr:\n%s", flagExtra.stderr)
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

func TestCLIEndToEndStopCountsAsOverride(t *testing.T) {
	t.Parallel()

	env := newCLIEnv(t)

	start := runCurfew(t, env, "", "start")
	if start.exitCode != 0 {
		t.Fatalf("start exit code = %d, stderr = %s", start.exitCode, start.stderr)
	}

	stop := runCurfew(t, env, "i am choosing to break my own rule\n", "stop")
	if stop.exitCode != 0 {
		t.Fatalf("stop exit code = %d, stderr = %s", stop.exitCode, stop.stderr)
	}
	if !strings.Contains(stop.stdout, "Curfew disabled") {
		t.Fatalf("expected stop output, got:\n%s", stop.stdout)
	}

	history := runCurfew(t, env, "", "history", "--days", "7")
	if history.exitCode != 0 {
		t.Fatalf("history exit code = %d, stderr = %s", history.exitCode, history.stderr)
	}
	if !strings.Contains(history.stdout, "overrode") {
		t.Fatalf("expected stop to count as an override, got:\n%s", history.stdout)
	}
}

func TestCLIInteractiveTUIEntrypoints(t *testing.T) {
	t.Parallel()

	env := newCLIEnv(t)

	rootSession := startCurfewPTY(t, env)
	waitForPTYText(t, rootSession, "Rules active:")
	stopCurfewPTY(t, rootSession, "q")

	configSession := startCurfewPTY(t, env, "config")
	waitForPTYText(t, configSession, "Default: bedtime 23:00 -> wake 07:00")
	stopCurfewPTY(t, configSession, "q")

	rulesSession := startCurfewPTY(t, env, "rules")
	waitForPTYText(t, rulesSession, "Default action: allow")
	stopCurfewPTY(t, rulesSession, "q")
}

func TestShellHooksEndToEnd(t *testing.T) {
	t.Parallel()
	if os.Getenv("CURFEW_RUN_PTY_E2E") == "" {
		t.Skip("set CURFEW_RUN_PTY_E2E=1 to run flaky PTY shell-hook coverage")
	}

	env := newCLIEnv(t)
	start := runCurfew(t, env, "", "start")
	if start.exitCode != 0 {
		t.Fatalf("start exit code = %d, stderr = %s", start.exitCode, start.stderr)
	}

	for _, shellKind := range []string{"zsh", "bash", "fish"} {
		shellKind := shellKind
		t.Run(shellKind, func(t *testing.T) {
			session := startShellPTY(t, env, shellKind)
			waitForPrompt(t, session)

			initPrompts := promptCount(session)
			writePTYLine(t, session, shellInitCommand(shellKind))
			waitForPromptCount(t, session, initPrompts+1)

			readyOffset := len(stripANSI(session.output.String()))
			readyPrompts := promptCount(session)
			writePTYLine(t, session, "echo ready")
			waitForPromptCount(t, session, readyPrompts+1)
			readyOutput := cleanedOutputAfter(session, readyOffset)
			if !strings.Contains(readyOutput, "ready") {
				t.Fatalf("expected allowed command output, got:\n%s", readyOutput)
			}

			startOffset := session.output.Len()
			blockedPrompts := promptCount(session)
			writePTYLine(t, session, "blocked-testcmd")
			waitForPTYTextAfter(t, session, "Type the passphrase to continue", startOffset)
			writePTYLine(t, session, "nope")
			waitForPromptCount(t, session, blockedPrompts+1)
			blockedOutput := stripANSI(session.output.String()[startOffset:])
			lower := strings.ToLower(blockedOutput)
			if strings.Contains(lower, "command not found") || strings.Contains(lower, "unknown command") {
				t.Fatalf("blocked command should not reach the shell, got:\n%s", blockedOutput)
			}

			afterOffset := len(stripANSI(session.output.String()))
			afterPrompts := promptCount(session)
			writePTYLine(t, session, "echo after")
			waitForPromptCount(t, session, afterPrompts+1)
			afterOutput := cleanedOutputAfter(session, afterOffset)
			if !strings.Contains(afterOutput, "after") {
				t.Fatalf("expected shell to remain usable after block, got:\n%s", afterOutput)
			}

			statusOffset := len(stripANSI(session.output.String()))
			statusPrompts := promptCount(session)
			writePTYLine(t, session, "curfew status")
			waitForPromptCount(t, session, statusPrompts+1)
			statusOutput := cleanedOutputAfter(session, statusOffset)
			if !strings.Contains(statusOutput, "Snoozes left:") {
				t.Fatalf("expected status output, got:\n%s", statusOutput)
			}

			stopCurfewPTY(t, session, "exit\n")
		})
	}
}

func newCLIEnv(t *testing.T) map[string]string {
	t.Helper()

	dir := t.TempDir()
	cfg := config.Default()
	cfg.Schedule.Timezone = "America/Los_Angeles"
	cfg.Rules.Rule = append(cfg.Rules.Rule, config.RuleEntry{Pattern: "blocked-testcmd", Action: "block"})

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

type ptySession struct {
	ptmx   *os.File
	output bytes.Buffer
	done   chan error
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

func startCurfewPTY(t *testing.T, env map[string]string, args ...string) *ptySession {
	t.Helper()

	command := exec.Command(mustBuildBinary(t), args...)
	command.Dir = repoRoot(t)

	environment := os.Environ()
	for key, value := range env {
		environment = append(environment, key+"="+value)
	}
	command.Env = environment

	ptmx, err := pty.Start(command)
	if err != nil {
		t.Fatalf("start pty: %v", err)
	}

	session := &ptySession{
		ptmx: ptmx,
		done: make(chan error, 1),
	}

	go func() {
		_, _ = io.Copy(&session.output, ptmx)
		session.done <- command.Wait()
	}()

	return session
}

func startShellPTY(t *testing.T, env map[string]string, shellKind string) *ptySession {
	t.Helper()

	shellPath, err := exec.LookPath(shellKind)
	if err != nil {
		t.Skipf("%s is not installed", shellKind)
	}

	args := []string{"-i"}
	command := exec.Command(shellPath, args...)
	command.Dir = repoRoot(t)

	environment := os.Environ()
	for key, value := range env {
		environment = append(environment, key+"="+value)
	}
	environment = append(environment,
		"PATH="+filepath.Dir(mustBuildBinary(t))+":"+os.Getenv("PATH"),
		"TERM=xterm-256color",
	)
	switch shellKind {
	case "zsh":
		zdotdir := t.TempDir()
		if err := os.WriteFile(filepath.Join(zdotdir, ".zshrc"), []byte("PROMPT='PROMPT> '\nRPROMPT=''\n"), 0o644); err != nil {
			t.Fatalf("write .zshrc: %v", err)
		}
		environment = append(environment, "ZDOTDIR="+zdotdir)
	case "bash":
		rcPath := filepath.Join(t.TempDir(), ".bashrc")
		if err := os.WriteFile(rcPath, []byte("PS1='PROMPT> '\n"), 0o644); err != nil {
			t.Fatalf("write .bashrc: %v", err)
		}
		command.Args = []string{shellPath, "--noprofile", "--rcfile", rcPath, "-i"}
	case "fish":
		configHome := env["XDG_CONFIG_HOME"]
		if err := os.MkdirAll(filepath.Join(configHome, "fish"), 0o755); err != nil {
			t.Fatalf("mkdir fish config: %v", err)
		}
		configText := "set -g fish_greeting\nfunction fish_prompt\n  printf 'PROMPT> '\nend\n"
		if err := os.WriteFile(filepath.Join(configHome, "fish", "config.fish"), []byte(configText), 0o644); err != nil {
			t.Fatalf("write fish config: %v", err)
		}
	}
	command.Env = environment

	ptmx, err := pty.Start(command)
	if err != nil {
		t.Fatalf("start shell pty: %v", err)
	}

	session := &ptySession{
		ptmx: ptmx,
		done: make(chan error, 1),
	}

	go func() {
		_, _ = io.Copy(&session.output, ptmx)
		session.done <- command.Wait()
	}()

	return session
}

func shellInitCommand(kind string) string {
	switch kind {
	case "fish":
		return "curfew init fish | source"
	default:
		return `eval "$(curfew init ` + kind + `)"`
	}
}

func waitForPTYText(t *testing.T, session *ptySession, want string) string {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		clean := stripANSI(session.output.String())
		if strings.Contains(clean, want) {
			return clean
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q in PTY output:\n%s", want, stripANSI(session.output.String()))
	return ""
}

func stopCurfewPTY(t *testing.T, session *ptySession, input string) {
	t.Helper()

	writePTY(t, session, input)
	select {
	case err := <-session.done:
		_ = session.ptmx.Close()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 0 {
				return
			}
			t.Fatalf("wait pty command: %v\noutput:\n%s", err, stripANSI(session.output.String()))
		}
	case <-time.After(5 * time.Second):
		_ = session.ptmx.Close()
		t.Fatalf("timed out waiting for PTY command to exit\noutput:\n%s", stripANSI(session.output.String()))
	}
}

func writePTY(t *testing.T, session *ptySession, input string) {
	t.Helper()

	if _, err := session.ptmx.Write([]byte(input)); err != nil {
		t.Fatalf("write pty input: %v", err)
	}
}

func writePTYLine(t *testing.T, session *ptySession, input string) {
	t.Helper()
	for _, char := range input {
		writePTY(t, session, string(char))
		time.Sleep(5 * time.Millisecond)
	}
	writePTY(t, session, "\n")
}

func waitForPrompt(t *testing.T, session *ptySession) string {
	t.Helper()
	return waitForPTYText(t, session, "PROMPT>")
}

func promptCount(session *ptySession) int {
	return strings.Count(stripANSI(session.output.String()), "PROMPT>")
}

func waitForPromptCount(t *testing.T, session *ptySession, want int) string {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		clean := stripANSI(session.output.String())
		if strings.Count(clean, "PROMPT>") >= want {
			return clean
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for prompt count %d in PTY output:\n%s", want, stripANSI(session.output.String()))
	return ""
}

func waitForPTYTextAfter(t *testing.T, session *ptySession, want string, offset int) string {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		full := session.output.String()
		if offset > len(full) {
			offset = len(full)
		}
		if strings.Contains(full[offset:], want) {
			return stripANSI(full)
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q in PTY output after %d bytes:\n%s", want, offset, stripANSI(session.output.String()))
	return ""
}

func cleanedOutputAfter(session *ptySession, offset int) string {
	clean := stripANSI(session.output.String())
	if offset > len(clean) {
		offset = len(clean)
	}
	return clean[offset:]
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

var ansiPattern = regexp.MustCompile(`\x1b(?:\[[0-9;?]*[ -/]*[@-~]|\][^\a]*(?:\a|\x1b\\))`)

func stripANSI(value string) string {
	clean := ansiPattern.ReplaceAllString(value, "")
	clean = strings.ReplaceAll(clean, "\r", "")
	return clean
}
