package rules

import (
	"testing"

	"github.com/rajjoshi/curfew/internal/config"
)

func TestEvaluate(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.Rules.Rule = []config.RuleEntry{
		{Pattern: "git push*", Action: "warn"},
		{Pattern: "git*", Action: "block"},
		{Pattern: "claude", Action: "block"},
		{Pattern: "aider*", Action: "block"},
	}

	tests := []struct {
		name            string
		command         string
		wantAction      string
		wantMatched     bool
		wantAllowlist   bool
		wantPattern     string
		wantCommandWord string
	}{
		{
			name:            "allowlist bypasses rule checks",
			command:         "ls -la",
			wantAction:      "allow",
			wantAllowlist:   true,
			wantCommandWord: "ls",
		},
		{
			name:            "exact word rule does not match suffixes",
			command:         "claude-code",
			wantAction:      "allow",
			wantCommandWord: "claude-code",
		},
		{
			name:            "exact word rule matches command word",
			command:         "claude",
			wantAction:      "block",
			wantMatched:     true,
			wantPattern:     "claude",
			wantCommandWord: "claude",
		},
		{
			name:            "glob prefix matches subcommands",
			command:         "aider --model gpt-5",
			wantAction:      "block",
			wantMatched:     true,
			wantPattern:     "aider*",
			wantCommandWord: "aider",
		},
		{
			name:            "subcommand prefix rule matches push",
			command:         "git push origin main",
			wantAction:      "warn",
			wantMatched:     true,
			wantPattern:     "git push*",
			wantCommandWord: "git",
		},
		{
			name:            "first match wins over generic glob",
			command:         "git push --force",
			wantAction:      "warn",
			wantMatched:     true,
			wantPattern:     "git push*",
			wantCommandWord: "git",
		},
		{
			name:            "env assignment still finds command word",
			command:         "FOO=1 claude",
			wantAction:      "block",
			wantMatched:     true,
			wantPattern:     "claude",
			wantCommandWord: "claude",
		},
		{
			name:            "other git subcommands do not match git push",
			command:         "git pull origin main",
			wantAction:      "block",
			wantMatched:     true,
			wantPattern:     "git*",
			wantCommandWord: "git",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			match := Evaluate(cfg, test.command)
			if match.Action != test.wantAction {
				t.Fatalf("action = %q, want %q", match.Action, test.wantAction)
			}
			if match.Matched != test.wantMatched {
				t.Fatalf("matched = %t, want %t", match.Matched, test.wantMatched)
			}
			if match.AllowedByAllowlist != test.wantAllowlist {
				t.Fatalf("allowlist = %t, want %t", match.AllowedByAllowlist, test.wantAllowlist)
			}
			if match.Pattern != test.wantPattern {
				t.Fatalf("pattern = %q, want %q", match.Pattern, test.wantPattern)
			}
			if match.CommandWord != test.wantCommandWord {
				t.Fatalf("command word = %q, want %q", match.CommandWord, test.wantCommandWord)
			}
		})
	}
}
