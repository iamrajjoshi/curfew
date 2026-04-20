package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rajjoshi/curfew/internal/app"
	"github.com/rajjoshi/curfew/internal/config"
	"github.com/rajjoshi/curfew/internal/paths"
	"github.com/rajjoshi/curfew/internal/runtime"
	"github.com/rajjoshi/curfew/internal/store"
)

func TestModelSaveShortcutPersistsDraft(t *testing.T) {
	t.Parallel()

	current := loadedTestModel(t)
	current.draft.Schedule.Default.Bedtime = "22:15"
	current.syncDraftState()

	current = runKeyMsg(t, current, tea.KeyMsg{Type: tea.KeyCtrlS})

	saved, err := current.app.LoadConfig()
	if err != nil {
		t.Fatalf("load config after save: %v", err)
	}
	if saved.Schedule.Default.Bedtime != "22:15" {
		t.Fatalf("saved bedtime = %q, want 22:15", saved.Schedule.Default.Bedtime)
	}
	if current.dirty {
		t.Fatal("expected save to clear dirty state")
	}
}

func TestModelSaveShortcutRejectsInvalidDraft(t *testing.T) {
	t.Parallel()

	current := loadedTestModel(t)
	current.draft.Schedule.Default.Bedtime = "nope"
	current.syncDraftState()

	updated, _ := current.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	current = updated.(model)

	saved, err := current.app.LoadConfig()
	if err != nil {
		t.Fatalf("load config after rejected save: %v", err)
	}
	if saved.Schedule.Default.Bedtime != "23:00" {
		t.Fatalf("saved bedtime = %q, want original value", saved.Schedule.Default.Bedtime)
	}
	if !strings.Contains(current.flash, "Fix validation errors") {
		t.Fatalf("expected validation flash, got %q", current.flash)
	}
}

func TestModelRevertShortcutRestoresPersistedConfig(t *testing.T) {
	t.Parallel()

	current := loadedTestModel(t)
	current.draft.Schedule.Default.Bedtime = "22:15"
	current.syncDraftState()

	current = runKeyMsg(t, current, tea.KeyMsg{Type: tea.KeyCtrlR})

	if current.draft.Schedule.Default.Bedtime != "23:00" {
		t.Fatalf("draft bedtime after revert = %q, want 23:00", current.draft.Schedule.Default.Bedtime)
	}
	if current.dirty {
		t.Fatal("expected revert to clear dirty state")
	}
}

func TestModelRefreshKeepsDirtyDraft(t *testing.T) {
	t.Parallel()

	current := loadedTestModel(t)
	current.draft.Schedule.Default.Bedtime = "22:15"
	current.syncDraftState()

	current = runKeyMsg(t, current, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	if current.draft.Schedule.Default.Bedtime != "22:15" {
		t.Fatalf("draft bedtime after refresh = %q, want 22:15", current.draft.Schedule.Default.Bedtime)
	}
	if !current.dirty {
		t.Fatal("expected refresh to keep draft dirty state")
	}
}

func TestRulesPreviewUsesDraftConfig(t *testing.T) {
	t.Parallel()

	current := loadedTestModel(t)
	current.activeTab = tabIndex("rules")
	current.draft.Rules.Rule = append(current.draft.Rules.Rule, config.RuleEntry{Pattern: "terraform plan*", Action: "warn"})
	current.syncDraftState()
	current.rulesTab.probeInput.SetValue("terraform plan example")

	output := current.rulesTab.view(current)
	if !strings.Contains(output, `Matched rule "terraform plan*"`) {
		t.Fatalf("expected draft preview to use unsaved rules, got:\n%s", output)
	}
}

func loadedTestModel(t *testing.T) model {
	t.Helper()

	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	now := time.Date(2026, 4, 22, 23, 30, 0, 0, location)
	application := newTestApp(t, now)
	current := newModel(application, "dashboard")
	message := current.Init()()
	updated, _ := current.Update(message)
	return updated.(model)
}

func newTestApp(t *testing.T, now time.Time) *app.App {
	t.Helper()

	dir := t.TempDir()
	layout := paths.Layout{
		Home:       dir,
		ConfigHome: filepath.Join(dir, ".config"),
		DataHome:   filepath.Join(dir, ".local", "share"),
		StateHome:  filepath.Join(dir, ".local", "state"),
	}
	if err := layout.Ensure(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}

	cfg := config.Default()
	cfg.Schedule.Timezone = "America/Los_Angeles"
	if err := config.Save(layout.ConfigFile(), cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	sqliteStore, err := store.Open(layout.HistoryDB())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		_ = sqliteStore.Close()
	})

	return &app.App{
		Paths:   layout,
		Runtime: runtime.New(layout.RuntimeFile(), layout.RuntimeLockFile()),
		Store:   sqliteStore,
		Now: func() time.Time {
			return now
		},
	}
}

func runKeyMsg(t *testing.T, current model, msg tea.KeyMsg) model {
	t.Helper()

	updated, cmd := current.Update(msg)
	next := updated.(model)
	return drainCmd(t, next, cmd)
}

func drainCmd(t *testing.T, current model, cmd tea.Cmd) model {
	t.Helper()

	for cmd != nil {
		msg := cmd()
		updated, nextCmd := current.Update(msg)
		current = updated.(model)
		cmd = nextCmd
	}
	return current
}
