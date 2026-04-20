package tui

import (
	"strings"
	"testing"

	"github.com/rajjoshi/curfew/internal/app"
	"github.com/rajjoshi/curfew/internal/config"
	"github.com/rajjoshi/curfew/internal/store"
)

func TestRenderRulesPreview(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	output := renderRules(cfg, cfg.Rules.Rule, "git push origin main")
	if !strings.Contains(output, `Matched rule "git push*"`) {
		t.Fatalf("expected rule preview to mention git push, got:\n%s", output)
	}
}

func TestDashboardSparklineAndStatus(t *testing.T) {
	t.Parallel()

	status := app.Status{SnoozesLeft: 2, Timezone: "America/Los_Angeles"}
	history := []store.HistoryRecord{
		{Status: "respected"},
		{Status: "snoozed"},
		{Status: "overrode"},
	}

	output := renderDashboard(status, []config.RuleEntry{{Pattern: "claude", Action: "block"}}, history)
	if !strings.Contains(output, "Rules active: 1") {
		t.Fatalf("expected dashboard to include the rules count, got:\n%s", output)
	}
	if !strings.Contains(output, "#~x") {
		t.Fatalf("expected dashboard to include the recent-history sparkline, got:\n%s", output)
	}
}
