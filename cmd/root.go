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
			currentShell := shell.Detect("", os.Getenv("SHELL"))
			rcPath, installed, err := shell.Installed(application.Paths, currentShell)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Current shell: %s\n", currentShell)
			fmt.Fprintf(cmd.OutOrStdout(), "Managed block installed: %t (%s)\n", installed, rcPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Hook active in current shell: %t\n", os.Getenv("CURFEW_SHELL_HOOK") == "1")
			fmt.Fprintf(cmd.OutOrStdout(), "Config file: %s (%t)\n", application.Paths.ConfigFile(), application.HasConfig())
			fmt.Fprintf(cmd.OutOrStdout(), "History DB: %s\n", application.Paths.HistoryDB())
			fmt.Fprintf(cmd.OutOrStdout(), "Runtime state: %s\n", application.Paths.RuntimeFile())

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

func newHistoryCmd(application *app.App) *cobra.Command {
	var days int
	command := &cobra.Command{
		Use:   "history",
		Short: "Show recent adherence history",
		RunE: func(cmd *cobra.Command, args []string) error {
			records, err := application.History(context.Background(), days)
			if err != nil {
				return err
			}
			if len(records) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No history yet.")
				return nil
			}
			for _, record := range records {
				fmt.Fprintf(cmd.OutOrStdout(), "%s  %-9s snoozes=%d overrides=%d blocked=%d\n", record.Date, record.Status, record.SnoozesUsed, record.Overrides, record.BlockedCount)
			}
			return nil
		},
	}
	command.Flags().IntVar(&days, "days", 7, "number of days to show")
	return command
}

func newStatsCmd(application *app.App) *cobra.Command {
	var days int
	command := &cobra.Command{
		Use:   "stats",
		Short: "Show numeric adherence stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			stats, err := application.Stats(context.Background(), days)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Respected nights: %d\n", stats.RespectedNights)
			fmt.Fprintf(cmd.OutOrStdout(), "Snoozed nights: %d\n", stats.SnoozedNights)
			fmt.Fprintf(cmd.OutOrStdout(), "Overridden nights: %d\n", stats.OverriddenNights)
			fmt.Fprintf(cmd.OutOrStdout(), "Current streak: %d\n", stats.CurrentStreak)
			fmt.Fprintf(cmd.OutOrStdout(), "Longest streak: %d\n", stats.LongestStreak)
			if stats.MostAttemptedCommand != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Most-attempted after-hours command: %s (%d)\n", stats.MostAttemptedCommand, stats.MostAttemptedCount)
			}
			return nil
		},
	}
	command.Flags().IntVar(&days, "days", 30, "number of days to summarize")
	return command
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
