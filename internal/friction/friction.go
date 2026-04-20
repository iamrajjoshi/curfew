package friction

import (
	"bufio"
	"fmt"
	"io"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/rajjoshi/curfew/internal/config"
	"github.com/rajjoshi/curfew/internal/schedule"
)

type Kind string

const (
	KindAllow      Kind = "allow"
	KindBanner     Kind = "banner"
	KindPrompt     Kind = "prompt"
	KindWait       Kind = "wait"
	KindPassphrase Kind = "passphrase"
	KindMath       Kind = "math"
	KindCombined   Kind = "combined"
	KindBlock      Kind = "block"
)

type Profile struct {
	Kind           Kind   `json:"kind"`
	WaitSeconds    int    `json:"wait_seconds,omitempty"`
	Passphrase     string `json:"passphrase,omitempty"`
	MathDifficulty string `json:"math_difficulty,omitempty"`
}

type IO struct {
	In    io.Reader
	Out   io.Writer
	Sleep func(time.Duration)
}

func EffectiveProfile(cfg config.Config, ruleAction string, tier schedule.Tier) Profile {
	if tier == schedule.TierNormal || ruleAction == "allow" {
		return Profile{Kind: KindAllow}
	}

	base := compilePreset(cfg, tier)
	switch ruleAction {
	case "warn":
		return stronger(base, Profile{Kind: KindPrompt})
	case "delay":
		return stronger(base, Profile{Kind: KindWait, WaitSeconds: 15})
	default:
		return base
	}
}

func Run(profile Profile, ioState IO, reason string) (bool, string, error) {
	if ioState.In == nil {
		ioState.In = strings.NewReader("")
	}
	if ioState.Out == nil {
		ioState.Out = io.Discard
	}
	if ioState.Sleep == nil {
		ioState.Sleep = time.Sleep
	}

	switch profile.Kind {
	case KindAllow:
		return true, "allowed", nil
	case KindBanner:
		fmt.Fprintf(ioState.Out, "%s\n", reason)
		return true, "allowed", nil
	case KindPrompt:
		fmt.Fprintf(ioState.Out, "%s\nProceed anyway? [y/N]: ", reason)
		answer, err := readLine(ioState.In)
		if err != nil {
			return false, "blocked", err
		}
		return isYes(answer), ternary(isYes(answer), "overridden", "blocked"), nil
	case KindWait:
		wait := max(profile.WaitSeconds, 1)
		fmt.Fprintf(ioState.Out, "%s\nWaiting %ds. Press Ctrl-C if this was accidental.\n", reason, wait)
		ioState.Sleep(time.Duration(wait) * time.Second)
		return true, "overridden", nil
	case KindPassphrase:
		fmt.Fprintf(ioState.Out, "%s\nType the passphrase to continue:\n> ", reason)
		answer, err := readLine(ioState.In)
		if err != nil {
			return false, "blocked", err
		}
		ok := strings.TrimSpace(answer) == strings.TrimSpace(profile.Passphrase)
		return ok, ternary(ok, "overridden", "blocked"), nil
	case KindMath:
		problem, answer := generateProblem(profile.MathDifficulty)
		fmt.Fprintf(ioState.Out, "%s\nSolve this to continue: %s = ", reason, problem)
		raw, err := readLine(ioState.In)
		if err != nil {
			return false, "blocked", err
		}
		value, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return false, "blocked", nil
		}
		ok := value == answer
		return ok, ternary(ok, "overridden", "blocked"), nil
	case KindCombined:
		wait := max(profile.WaitSeconds, 1)
		fmt.Fprintf(ioState.Out, "%s\nWaiting %ds before the override prompt.\n", reason, wait)
		ioState.Sleep(time.Duration(wait) * time.Second)
		next := Profile{
			Kind:           KindPassphrase,
			Passphrase:     profile.Passphrase,
			MathDifficulty: profile.MathDifficulty,
		}
		if strings.TrimSpace(next.Passphrase) == "" {
			next.Kind = KindMath
		}
		return Run(next, ioState, "Override still requested.")
	case KindBlock:
		fmt.Fprintf(ioState.Out, "%s\n", reason)
		return false, "blocked", nil
	default:
		return false, "blocked", fmt.Errorf("unknown friction profile %q", profile.Kind)
	}
}

func compilePreset(cfg config.Config, tier schedule.Tier) Profile {
	switch strings.ToLower(cfg.Override.Preset) {
	case "soft":
		switch tier {
		case schedule.TierWarning:
			return Profile{Kind: KindBanner}
		case schedule.TierCurfew, schedule.TierHardStop:
			return Profile{Kind: KindPrompt}
		}
	case "medium":
		switch tier {
		case schedule.TierWarning:
			return Profile{Kind: KindPrompt}
		case schedule.TierCurfew:
			return Profile{Kind: KindPassphrase, Passphrase: cfg.Override.Passphrase}
		case schedule.TierHardStop:
			return Profile{Kind: KindBlock}
		}
	case "hard":
		switch tier {
		case schedule.TierWarning:
			return Profile{Kind: KindWait, WaitSeconds: 30}
		case schedule.TierCurfew:
			return Profile{Kind: KindCombined, Passphrase: cfg.Override.Passphrase, WaitSeconds: max(cfg.Override.WaitSeconds, 60)}
		case schedule.TierHardStop:
			return Profile{Kind: KindBlock}
		}
	case "custom":
		return compileCustom(cfg, tier)
	}
	return Profile{Kind: KindPassphrase, Passphrase: cfg.Override.Passphrase}
}

func compileCustom(cfg config.Config, tier schedule.Tier) Profile {
	var source config.OverrideTier
	switch tier {
	case schedule.TierWarning:
		source = cfg.Override.Custom.Warning
	case schedule.TierCurfew:
		source = cfg.Override.Custom.Curfew
	case schedule.TierHardStop:
		source = cfg.Override.Custom.HardStop
	default:
		return Profile{Kind: KindAllow}
	}

	switch strings.ToLower(strings.TrimSpace(source.Action)) {
	case "allow":
		return Profile{Kind: KindAllow}
	case "block":
		return Profile{Kind: KindBlock}
	}

	switch strings.ToLower(strings.TrimSpace(source.Method)) {
	case "", "none":
		return Profile{Kind: KindAllow}
	case "prompt":
		return Profile{Kind: KindPrompt}
	case "wait":
		return Profile{Kind: KindWait, WaitSeconds: max(source.WaitSeconds, cfg.Override.WaitSeconds)}
	case "passphrase":
		return Profile{Kind: KindPassphrase, Passphrase: coalesce(source.Passphrase, cfg.Override.Passphrase)}
	case "math":
		return Profile{Kind: KindMath, MathDifficulty: coalesce(source.MathDifficulty, cfg.Override.MathDifficulty)}
	case "combined":
		return Profile{
			Kind:           KindCombined,
			Passphrase:     coalesce(source.Passphrase, cfg.Override.Passphrase),
			WaitSeconds:    max(source.WaitSeconds, cfg.Override.WaitSeconds),
			MathDifficulty: coalesce(source.MathDifficulty, cfg.Override.MathDifficulty),
		}
	default:
		return Profile{Kind: KindBlock}
	}
}

func stronger(left Profile, right Profile) Profile {
	if rank(right.Kind) > rank(left.Kind) {
		switch right.Kind {
		case KindWait:
			right.WaitSeconds = max(right.WaitSeconds, left.WaitSeconds)
		case KindPassphrase, KindCombined:
			if right.Passphrase == "" {
				right.Passphrase = left.Passphrase
			}
		}
		return right
	}
	return left
}

func rank(kind Kind) int {
	switch kind {
	case KindAllow:
		return 0
	case KindBanner:
		return 1
	case KindPrompt:
		return 2
	case KindWait:
		return 3
	case KindMath:
		return 4
	case KindPassphrase:
		return 5
	case KindCombined:
		return 6
	case KindBlock:
		return 7
	default:
		return 99
	}
}

func generateProblem(difficulty string) (string, int) {
	switch strings.ToLower(strings.TrimSpace(difficulty)) {
	case "easy":
		a := rand.IntN(90) + 10
		b := rand.IntN(90) + 10
		return fmt.Sprintf("%d + %d", a, b), a + b
	case "hard":
		a := rand.IntN(15) + 2
		b := rand.IntN(11) + 2
		m := rand.IntN(17) + 3
		answer := (a * b) % m
		return fmt.Sprintf("(%d * %d) mod %d", a, b, m), answer
	default:
		a := rand.IntN(900) + 100
		b := rand.IntN(90) + 10
		return fmt.Sprintf("%d * %d", a, b), a * b
	}
}

func readLine(input io.Reader) (string, error) {
	reader := bufio.NewReader(input)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func isYes(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "y" || value == "yes"
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func ternary[T any](condition bool, left T, right T) T {
	if condition {
		return left
	}
	return right
}
