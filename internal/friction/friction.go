package friction

import (
	"bufio"
	"fmt"
	"io"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/iamrajjoshi/curfew/internal/config"
	"github.com/iamrajjoshi/curfew/internal/schedule"
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

type Challenge struct {
	Kind        Kind
	Reason      string
	WaitSeconds int
	Prompt      string
	Help        string

	passphrase string
	answer     string
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

func NewChallenge(profile Profile, reason string) Challenge {
	challenge := Challenge{
		Kind:   profile.Kind,
		Reason: reason,
	}

	switch profile.Kind {
	case KindWait, KindCombined:
		challenge.WaitSeconds = max(profile.WaitSeconds, 1)
	}

	switch profile.Kind {
	case KindPrompt:
		challenge.Prompt = "Proceed anyway? [y/N]"
	case KindPassphrase:
		challenge.Prompt = "Type the passphrase to continue"
		challenge.Help = strings.TrimSpace(profile.Passphrase)
		challenge.passphrase = strings.TrimSpace(profile.Passphrase)
	case KindMath:
		problem, answer := generateProblem(profile.MathDifficulty)
		challenge.Prompt = "Solve this to continue"
		challenge.Help = fmt.Sprintf("%s =", problem)
		challenge.answer = strconv.Itoa(answer)
	case KindCombined:
		challenge.Kind = KindCombined
		if strings.TrimSpace(profile.Passphrase) != "" {
			challenge.Prompt = "Type the passphrase to continue"
			challenge.Help = strings.TrimSpace(profile.Passphrase)
			challenge.passphrase = strings.TrimSpace(profile.Passphrase)
			break
		}
		problem, answer := generateProblem(profile.MathDifficulty)
		challenge.Prompt = "Solve this to continue"
		challenge.Help = fmt.Sprintf("%s =", problem)
		challenge.answer = strconv.Itoa(answer)
	}

	return challenge
}

func (c Challenge) RequiresInput() bool {
	switch c.Kind {
	case KindPrompt, KindPassphrase, KindMath, KindCombined:
		return true
	default:
		return false
	}
}

func (c Challenge) Check(input string) (bool, string) {
	switch c.Kind {
	case KindAllow, KindBanner:
		return true, "allowed"
	case KindPrompt:
		ok := isYes(input)
		return ok, ternary(ok, "overridden", "blocked")
	case KindWait:
		return true, "overridden"
	case KindPassphrase, KindCombined:
		if c.passphrase != "" {
			ok := strings.TrimSpace(input) == c.passphrase
			return ok, ternary(ok, "overridden", "blocked")
		}
		fallthrough
	case KindMath:
		ok := strings.TrimSpace(input) == c.answer
		return ok, ternary(ok, "overridden", "blocked")
	case KindBlock:
		return false, "blocked"
	default:
		return false, "blocked"
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

	challenge := NewChallenge(profile, reason)

	switch challenge.Kind {
	case KindAllow:
		return true, "allowed", nil
	case KindBanner:
		fmt.Fprintf(ioState.Out, "%s\n", reason)
		return true, "allowed", nil
	case KindPrompt:
		fmt.Fprintf(ioState.Out, "%s\n%s: ", challenge.Reason, challenge.Prompt)
		answer, err := readLine(ioState.In)
		if err != nil {
			return false, "blocked", err
		}
		allowed, outcome := challenge.Check(answer)
		return allowed, outcome, nil
	case KindWait:
		fmt.Fprintf(ioState.Out, "%s\nWaiting %ds. Press Ctrl-C if this was accidental.\n", challenge.Reason, challenge.WaitSeconds)
		ioState.Sleep(time.Duration(challenge.WaitSeconds) * time.Second)
		return true, "overridden", nil
	case KindPassphrase:
		fmt.Fprintf(ioState.Out, "%s\n%s:\n> ", challenge.Reason, challenge.Prompt)
		answer, err := readLine(ioState.In)
		if err != nil {
			return false, "blocked", err
		}
		allowed, outcome := challenge.Check(answer)
		return allowed, outcome, nil
	case KindMath:
		fmt.Fprintf(ioState.Out, "%s\n%s: %s ", challenge.Reason, challenge.Prompt, challenge.Help)
		raw, err := readLine(ioState.In)
		if err != nil {
			return false, "blocked", err
		}
		allowed, outcome := challenge.Check(raw)
		return allowed, outcome, nil
	case KindCombined:
		fmt.Fprintf(ioState.Out, "%s\nWaiting %ds before the override prompt.\n", challenge.Reason, challenge.WaitSeconds)
		ioState.Sleep(time.Duration(challenge.WaitSeconds) * time.Second)
		if challenge.passphrase != "" {
			fmt.Fprintf(ioState.Out, "Override still requested.\n%s:\n> ", challenge.Prompt)
		} else {
			fmt.Fprintf(ioState.Out, "Override still requested.\n%s: %s ", challenge.Prompt, challenge.Help)
		}
		raw, err := readLine(ioState.In)
		if err != nil {
			return false, "blocked", err
		}
		allowed, outcome := challenge.Check(raw)
		return allowed, outcome, nil
	case KindBlock:
		fmt.Fprintf(ioState.Out, "%s\n", challenge.Reason)
		return false, "blocked", nil
	default:
		return false, "blocked", fmt.Errorf("unknown friction profile %q", challenge.Kind)
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
