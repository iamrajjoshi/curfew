package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rajjoshi/curfew/internal/paths"
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

func TestInitScriptsReferenceCurfewCheck(t *testing.T) {
	t.Parallel()

	for _, kind := range []string{"zsh", "bash", "fish"} {
		kind := kind
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			script, err := Init(kind)
			if err != nil {
				t.Fatalf("init: %v", err)
			}
			if !strings.Contains(script, "curfew check") {
				t.Fatalf("script does not reference curfew check:\n%s", script)
			}
			if !strings.Contains(script, "CURFEW_SHELL_HOOK") {
				t.Fatalf("script does not export the shell hook marker:\n%s", script)
			}
		})
	}
}
