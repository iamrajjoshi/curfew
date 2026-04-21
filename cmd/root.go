package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/iamrajjoshi/curfew/internal/app"
	"github.com/iamrajjoshi/curfew/internal/config"
	"github.com/iamrajjoshi/curfew/internal/schedule"
	"github.com/iamrajjoshi/curfew/internal/setup"
	"github.com/iamrajjoshi/curfew/internal/shell"
	"github.com/iamrajjoshi/curfew/internal/shim"
	"github.com/iamrajjoshi/curfew/internal/store"
	"github.com/iamrajjoshi/curfew/internal/tui"
)

func Execute() error {
	application, err := app.New()
	if err != nil {
		return err
	}
	defer application.Close()

	root := newRootCmd(application)
	return root.Execute()
}

func newRootCmd(application *app.App) *cobra.Command {
	var showVersion bool

	root := &cobra.Command{
		Use:           "curfew",
		Short:         "Protect your quiet hours from risky terminal commands",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Fprintln(cmd.OutOrStdout(), versionString())
				return nil
			}
			if !application.HasConfig() {
				if !isInteractiveSession() {
					return errors.New("curfew is not configured yet; run `curfew config edit` in a terminal")
				}
				cfg, err := setup.Run(application.DefaultConfig())
				if err != nil {
					return err
				}
				if err := application.SaveConfig(cfg); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Saved %s\n", application.Paths.ConfigFile())
				return nil
			}
			if isInteractiveSession() {
				return tui.Run(application, "dashboard")
			}
			status, err := application.EvaluateStatus()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), status.Render())
			return nil
		},
	}

	root.AddCommand(newStatusCmd(application))
	root.AddCommand(newCheckCmd(application))
	root.AddCommand(newStartCmd(application))
	root.AddCommand(newStopCmd(application))
	root.AddCommand(newSnoozeCmd(application))
	root.AddCommand(newSkipCmd(application))
	root.AddCommand(newRulesCmd(application))
	root.AddCommand(newInstallCmd(application))
	root.AddCommand(newUninstallCmd(application))
	root.AddCommand(newInitCmd())
	root.AddCommand(newShimCmd(application))
	root.AddCommand(newDoctorCmd(application))
	root.AddCommand(newHistoryCmd(application))
	root.AddCommand(newStatsCmd(application))
	root.AddCommand(newConfigCmd(application))
	root.AddCommand(newVersionCmd())
	root.Flags().BoolVarP(&showVersion, "version", "v", false, "version for curfew")

	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show curfew version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), versionString())
		},
	}
}

func newStatusCmd(application *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show curfew status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !application.HasConfig() {
				return errors.New("curfew is not configured yet; run `curfew` or `curfew config` first")
			}
			status, err := application.EvaluateStatus()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), status.Render())
			return nil
		},
	}
}

func newCheckCmd(application *app.App) *cobra.Command {
	var jsonMode bool
	var shellKind string

	command := &cobra.Command{
		Use:   "check <cmd>",
		Short: "Evaluate a command against the current curfew state",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			raw := strings.TrimSpace(strings.Join(args, " "))
			result, err := application.Check(context.Background(), raw, app.CheckOptions{
				Shell: shellKind,
				JSON:  jsonMode,
				In:    cmd.InOrStdin(),
				Out:   cmd.ErrOrStderr(),
			})
			if err != nil {
				return err
			}
			app.WriteCheckResult(result, jsonMode, cmd.ErrOrStderr())
			if result.Allowed {
				return nil
			}
			return exitError{code: 1}
		},
	}

	command.Flags().BoolVar(&jsonMode, "json", false, "print JSON output")
	command.Flags().StringVar(&shellKind, "shell", "", "shell backend that invoked curfew")
	return command
}

func newStartCmd(application *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Force-enable curfew now",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := application.SetForcedActive(context.Background())
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Curfew force-enabled for session %s until %s.\n", session.Date, session.Wake.Format("15:04"))
			return nil
		},
	}
}

func newStopCmd(application *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Disable curfew for the rest of this session",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := application.StopTonight(
				context.Background(),
				"Disable curfew for the rest of this session?",
				cmd.InOrStdin(),
				cmd.ErrOrStderr(),
			)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Curfew disabled for session %s.\n", session.Date)
			return nil
		},
	}
}

func newSnoozeCmd(application *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "snooze [duration]",
		Short: "Snooze curfew for a short period",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := application.LoadConfig()
			if err != nil {
				return err
			}
			duration, err := time.ParseDuration(cfg.Grace.SnoozeDuration)
			if err != nil {
				return err
			}
			if len(args) == 1 {
				duration, err = time.ParseDuration(args[0])
				if err != nil {
					return err
				}
			}
			session, snoozedUntil, remaining, err := application.Snooze(context.Background(), duration)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Curfew snoozed for session %s until %s. %d snoozes remaining.\n", session.Date, snoozedUntil.Format(time.RFC3339), remaining)
			return nil
		},
	}
}

func newSkipCmd(application *app.App) *cobra.Command {
	skip := &cobra.Command{
		Use:   "skip",
		Short: "Skip a curfew period",
	}
	skip.AddCommand(&cobra.Command{
		Use:   "tonight",
		Short: "Skip tonight's curfew",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := application.SkipTonight(
				context.Background(),
				"Skip tonight's curfew?",
				cmd.InOrStdin(),
				cmd.ErrOrStderr(),
			)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Skipped curfew for session %s.\n", session.Date)
			return nil
		},
	})
	return skip
}

func newRulesCmd(application *app.App) *cobra.Command {
	rulesCmd := &cobra.Command{
		Use:   "rules",
		Short: "Inspect or edit command rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			if isInteractiveSession() {
				return tui.Run(application, "rules")
			}
			return listRules(cmd, application)
		},
	}

	list := &cobra.Command{
		Use:   "list",
		Short: "List configured rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listRules(cmd, application)
		},
	}

	add := &cobra.Command{
		Use:   "add <pattern>",
		Short: "Add a new rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			action, _ := cmd.Flags().GetString("action")
			cfg, err := application.LoadConfig()
			if err != nil {
				return err
			}
			cfg.Rules.Rule = append(cfg.Rules.Rule, config.RuleEntry{Pattern: args[0], Action: action})
			if err := application.SaveConfig(cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added %s -> %s\n", args[0], action)
			return nil
		},
	}
	add.Flags().String("action", "block", "rule action: allow, block, warn, delay")

	rm := &cobra.Command{
		Use:   "rm <pattern>",
		Short: "Remove a rule by pattern",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			removed, err := application.RemoveRule(args[0])
			if err != nil {
				return err
			}
			if !removed {
				return fmt.Errorf("rule %q not found", args[0])
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", args[0])
			return nil
		},
	}

	rulesCmd.AddCommand(list, add, rm)
	return rulesCmd
}

func listRules(cmd *cobra.Command, application *app.App) error {
	rules, err := application.Rules()
	if err != nil {
		return err
	}
	for _, rule := range rules {
		fmt.Fprintf(cmd.OutOrStdout(), "%-6s %s\n", rule.Action, rule.Pattern)
	}
	return nil
}

func newInstallCmd(application *app.App) *cobra.Command {
	var explicitShell string
	command := &cobra.Command{
		Use:   "install",
		Short: "Install the shell hook into your rc file",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := shell.Detect(explicitShell, os.Getenv("SHELL"))
			if !shell.Supported(kind) {
				return fmt.Errorf("unsupported shell %q", kind)
			}
			rcPath, changed, err := shell.Install(application.Paths, kind)
			if err != nil {
				return err
			}
			if changed {
				fmt.Fprintf(cmd.OutOrStdout(), "Installed curfew into %s\n", rcPath)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Curfew is already installed in %s\n", rcPath)
			}
			return nil
		},
	}
	command.Flags().StringVarP(&explicitShell, "shell", "s", "", "shell to configure")
	return command
}

func newUninstallCmd(application *app.App) *cobra.Command {
	var explicitShell string
	command := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the shell hook from your rc file",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := shell.Detect(explicitShell, os.Getenv("SHELL"))
			if !shell.Supported(kind) {
				return fmt.Errorf("unsupported shell %q", kind)
			}
			rcPath, changed, err := shell.Uninstall(application.Paths, kind)
			if err != nil {
				return err
			}
			if changed {
				fmt.Fprintf(cmd.OutOrStdout(), "Removed curfew from %s\n", rcPath)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "No curfew block found in %s\n", rcPath)
			}
			return nil
		},
	}
	command.Flags().StringVarP(&explicitShell, "shell", "s", "", "shell to clean up")
	return command
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <shell>",
		Short: "Emit shell integration code for a specific shell",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			script, err := shell.Init(shell.Detect(args[0], ""))
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), script)
			return nil
		},
	}
}

func newDoctorCmd(application *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run quick diagnostics",
		RunE: func(cmd *cobra.Command, args []string) error {
			shellDiagnostics, err := shell.Diagnose(
				application.Paths,
				"",
				os.Getenv("SHELL"),
				os.Getenv("CURFEW_SHELL_KIND"),
				os.Getenv("CURFEW_SHELL_HOOK") == "1",
			)
			if err != nil {
				return err
			}
			shimConfig, err := loadShimConfig(application)
			if err != nil {
				return err
			}
			shimDiagnostics, err := shim.Diagnose(application.Paths, shellDiagnostics.DetectedShell, shimConfig)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Detected shell: %s\n", shellDiagnostics.DetectedShell)
			fmt.Fprintf(cmd.OutOrStdout(), "Managed rc/config path: %s\n", shellDiagnostics.RCPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Managed block installed: %t\n", shellDiagnostics.ManagedBlockInstalled)
			fmt.Fprintf(cmd.OutOrStdout(), "Hook active in current shell: %t\n", shellDiagnostics.HookActive)
			if shellDiagnostics.HookShell != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Hook shell kind: %s\n", shellDiagnostics.HookShell)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Hook shell kind: n/a")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Config file: %s (%t)\n", application.Paths.ConfigFile(), application.HasConfig())
			fmt.Fprintf(cmd.OutOrStdout(), "History DB: %s\n", application.Paths.HistoryDB())
			fmt.Fprintf(cmd.OutOrStdout(), "Runtime state: %s\n", application.Paths.RuntimeFile())
			fmt.Fprintf(cmd.OutOrStdout(), "Shim directory: %s\n", shimDiagnostics.ShimDir)
			fmt.Fprintf(cmd.OutOrStdout(), "Shim PATH block installed: %t\n", shimDiagnostics.PathBlockInstalled)
			fmt.Fprintf(cmd.OutOrStdout(), "Installed shims: %d\n", len(shimDiagnostics.InstalledCommands))

			if application.HasConfig() {
				cfg, err := application.LoadConfig()
				if err != nil {
					return err
				}
				location, err := schedule.ResolveLocation(cfg.Schedule.Timezone)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Configured timezone: %s -> %s\n", cfg.Schedule.Timezone, location.String())
			}
			return nil
		},
	}
}

func newShimCmd(application *app.App) *cobra.Command {
	var explicitShell string

	command := &cobra.Command{
		Use:   "shim",
		Short: "Manage PATH shims for non-interactive enforcement",
	}

	command.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Install curfew PATH shims into your shell config",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := shell.Detect(explicitShell, os.Getenv("SHELL"))
			if !shell.Supported(kind) {
				return fmt.Errorf("unsupported shell %q", kind)
			}
			cfg, err := loadShimConfig(application)
			if err != nil {
				return err
			}
			result, err := shim.Install(application.Paths, kind, cfg)
			if err != nil {
				return err
			}
			if result.PathChanged {
				fmt.Fprintf(cmd.OutOrStdout(), "Installed %d shims in %s and updated %s\n", len(result.Installed), result.ShimDir, result.RCPath)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Installed %d shims in %s. Shim PATH block already existed in %s\n", len(result.Installed), result.ShimDir, result.RCPath)
			}
			return nil
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "uninstall",
		Short: "Remove curfew PATH shims from your shell config",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := shell.Detect(explicitShell, os.Getenv("SHELL"))
			if !shell.Supported(kind) {
				return fmt.Errorf("unsupported shell %q", kind)
			}
			result, err := shim.Uninstall(application.Paths, kind)
			if err != nil {
				return err
			}
			if result.PathChanged {
				fmt.Fprintf(cmd.OutOrStdout(), "Removed %d shims from %s and cleaned %s\n", result.Removed, result.ShimDir, result.RCPath)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Removed %d shims from %s. No shim PATH block found in %s\n", result.Removed, result.ShimDir, result.RCPath)
			}
			return nil
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show curfew shim installation status",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := shell.Detect(explicitShell, os.Getenv("SHELL"))
			if !shell.Supported(kind) {
				return fmt.Errorf("unsupported shell %q", kind)
			}
			cfg, err := loadShimConfig(application)
			if err != nil {
				return err
			}
			diagnostics, err := shim.Diagnose(application.Paths, kind, cfg)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Detected shell: %s\n", kind)
			fmt.Fprintf(cmd.OutOrStdout(), "Managed rc/config path: %s\n", diagnostics.RCPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Shim directory: %s\n", diagnostics.ShimDir)
			fmt.Fprintf(cmd.OutOrStdout(), "Shim PATH block installed: %t\n", diagnostics.PathBlockInstalled)
			fmt.Fprintf(cmd.OutOrStdout(), "Installed shims (%d): %s\n", len(diagnostics.InstalledCommands), renderStringList(diagnostics.InstalledCommands))
			fmt.Fprintf(cmd.OutOrStdout(), "Expected shims (%d): %s\n", len(diagnostics.ExpectedCommands), renderStringList(diagnostics.ExpectedCommands))
			fmt.Fprintf(cmd.OutOrStdout(), "Missing shims: %s\n", renderStringList(diagnostics.MissingCommands))
			fmt.Fprintf(cmd.OutOrStdout(), "Extra shims: %s\n", renderStringList(diagnostics.ExtraCommands))
			return nil
		},
	})

	command.PersistentFlags().StringVarP(&explicitShell, "shell", "s", "", "shell to configure")
	return command
}

func loadShimConfig(application *app.App) (config.Config, error) {
	if !application.HasConfig() {
		return config.Config{}, nil
	}
	return application.LoadConfig()
}

func renderStringList(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func newHistoryCmd(application *app.App) *cobra.Command {
	var days int
	var jsonMode bool
	command := &cobra.Command{
		Use:   "history",
		Short: "Show recent adherence history",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			records, err := application.History(context.Background(), days)
			if err != nil {
				return err
			}
			if jsonMode {
				fmt.Fprintln(cmd.OutOrStdout(), application.JSON(records))
				return nil
			}
			if len(records) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No history yet.")
				return nil
			}
			for _, record := range records {
				fmt.Fprintln(cmd.OutOrStdout(), renderHistorySummary(record))
			}
			return nil
		},
	}
	command.Flags().IntVar(&days, "days", 7, "number of days to show")
	command.Flags().BoolVar(&jsonMode, "json", false, "print JSON output")
	command.AddCommand(newHistoryShowCmd(application))
	return command
}

func newStatsCmd(application *app.App) *cobra.Command {
	var days int
	var jsonMode bool
	command := &cobra.Command{
		Use:   "stats",
		Short: "Show numeric adherence stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			stats, err := application.Stats(context.Background(), days)
			if err != nil {
				return err
			}
			if jsonMode {
				payload := struct {
					WindowDays int `json:"window_days"`
					store.Stats
				}{
					WindowDays: days,
					Stats:      stats,
				}
				fmt.Fprintln(cmd.OutOrStdout(), application.JSON(payload))
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Total nights: %d\n", stats.TotalNights)
			fmt.Fprintf(cmd.OutOrStdout(), "Respected nights: %d\n", stats.RespectedNights)
			fmt.Fprintf(cmd.OutOrStdout(), "Snoozed nights: %d\n", stats.SnoozedNights)
			fmt.Fprintf(cmd.OutOrStdout(), "Overridden nights: %d\n", stats.OverriddenNights)
			fmt.Fprintf(cmd.OutOrStdout(), "Adherent nights: %d\n", stats.AdherentNights)
			fmt.Fprintf(cmd.OutOrStdout(), "Adherence rate: %.1f%%\n", stats.AdherenceRate*100)
			fmt.Fprintf(cmd.OutOrStdout(), "Current streak: %d\n", stats.CurrentStreak)
			fmt.Fprintf(cmd.OutOrStdout(), "Longest streak: %d\n", stats.LongestStreak)
			if len(stats.TopCommands) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Top after-hours commands:")
				for _, command := range stats.TopCommands {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s (%d)\n", command.Command, command.Count)
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Top after-hours commands: none")
			}
			return nil
		},
	}
	command.Flags().IntVar(&days, "days", 30, "number of days to summarize")
	command.Flags().BoolVar(&jsonMode, "json", false, "print JSON output")
	return command
}

func newHistoryShowCmd(application *app.App) *cobra.Command {
	var jsonMode bool

	command := &cobra.Command{
		Use:   "show <YYYY-MM-DD>",
		Short: "Show detailed history for one curfew night",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := time.Parse("2006-01-02", args[0]); err != nil {
				return fmt.Errorf("invalid session date %q; expected YYYY-MM-DD", args[0])
			}

			details, err := application.HistoryDetails(context.Background(), args[0])
			if err != nil {
				return err
			}
			if jsonMode {
				fmt.Fprintln(cmd.OutOrStdout(), application.JSON(details))
				return nil
			}
			if !details.Found {
				fmt.Fprintf(cmd.OutOrStdout(), "No history for %s.\n", args[0])
				return nil
			}
			fmt.Fprint(cmd.OutOrStdout(), renderHistoryDetails(details))
			return nil
		},
	}

	command.Flags().BoolVar(&jsonMode, "json", false, "print JSON output")
	return command
}

func renderHistorySummary(record store.HistoryRecord) string {
	lastCommand := "n/a"
	if record.LastCommand != nil {
		lastCommand = record.LastCommand.Format(time.RFC3339)
	}
	return fmt.Sprintf(
		"%s  %-9s snoozes=%d overrides=%d blocked=%d last=%s",
		record.Date,
		record.Status,
		record.SnoozesUsed,
		record.Overrides,
		record.BlockedCount,
		lastCommand,
	)
}

func renderHistoryDetails(details store.SessionDetails) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("Date: %s", details.Session.Date))
	lines = append(lines, fmt.Sprintf("Status: %s", details.Session.Status))
	lines = append(lines, fmt.Sprintf("Configured schedule: %s -> %s", details.Session.BedtimeConfigured, details.Session.WakeConfigured))
	lines = append(lines, fmt.Sprintf("Snoozes used: %d", details.Session.SnoozesUsed))
	lines = append(lines, fmt.Sprintf("Skipped: %t", details.Session.Skipped))
	lines = append(lines, fmt.Sprintf("Forced active: %t", details.Session.ForcedActive))
	lines = append(lines, fmt.Sprintf("Blocked attempts: %d", details.Session.BlockedCount))
	lines = append(lines, fmt.Sprintf("Overrides: %d", details.Session.Overrides))
	if details.Session.FirstBlockedAt != nil {
		lines = append(lines, fmt.Sprintf("First blocked at: %s", details.Session.FirstBlockedAt.Format(time.RFC3339)))
	}
	if details.Session.LastCommandAt != nil {
		lines = append(lines, fmt.Sprintf("Last command at: %s", details.Session.LastCommandAt.Format(time.RFC3339)))
	}
	lines = append(lines, "", "Events", "------")
	if len(details.Events) == 0 {
		lines = append(lines, "No intercepted commands.")
		return strings.Join(lines, "\n") + "\n"
	}
	for _, event := range details.Events {
		line := fmt.Sprintf("%s  %s  action=%s outcome=%s", event.Timestamp.Format(time.RFC3339), event.Command, event.Action, event.Outcome)
		if event.Tier != "" {
			line += fmt.Sprintf(" tier=%s", event.Tier)
		}
		if event.MatchedRule != "" {
			line += fmt.Sprintf(" rule=%q", event.MatchedRule)
		}
		if event.Shell != "" {
			line += fmt.Sprintf(" shell=%s", event.Shell)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n") + "\n"
}

func newConfigCmd(application *app.App) *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Open curfew configuration views",
		RunE: func(cmd *cobra.Command, args []string) error {
			if isInteractiveSession() && application.HasConfig() {
				return tui.Run(application, "schedule")
			}
			if !isInteractiveSession() && application.HasConfig() {
				fmt.Fprintf(cmd.OutOrStdout(), "Config file: %s\n", application.Paths.ConfigFile())
				return nil
			}
			current := application.DefaultConfig()
			if application.HasConfig() {
				existing, err := application.LoadConfig()
				if err != nil {
					return err
				}
				current = existing
			}
			updated, err := setup.Run(current)
			if err != nil {
				return err
			}
			if err := application.SaveConfig(updated); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Saved %s\n", application.Paths.ConfigFile())
			return nil
		},
	}

	configCmd.AddCommand(&cobra.Command{
		Use:   "edit",
		Short: "Run the interactive first-run style config editor",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isInteractiveSession() {
				return errors.New("config edit requires an interactive terminal")
			}
			current := application.DefaultConfig()
			if application.HasConfig() {
				existing, err := application.LoadConfig()
				if err != nil {
					return err
				}
				current = existing
			}
			updated, err := setup.Run(current)
			if err != nil {
				return err
			}
			if err := application.SaveConfig(updated); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Saved %s\n", application.Paths.ConfigFile())
			return nil
		},
	})

	return configCmd
}

func isInteractiveSession() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
