package rules

import (
	"strings"

	"github.com/iamrajjoshi/curfew/internal/config"
)

type Kind string

const (
	KindExactWord   Kind = "exact_word"
	KindPrefixWords      = "prefix_words"
	KindGlobPrefix       = "glob_prefix"
)

type Compiled struct {
	Pattern      string
	Action       string
	Kind         Kind
	Fields       []string
	LastPrefix   string
	OriginalRule config.RuleEntry
}

type Match struct {
	AllowedByAllowlist bool   `json:"allowed_by_allowlist"`
	Matched            bool   `json:"matched"`
	Pattern            string `json:"pattern,omitempty"`
	Action             string `json:"action"`
	Kind               Kind   `json:"kind,omitempty"`
	CommandWord        string `json:"command_word,omitempty"`
}

type Command struct {
	Raw         string
	Fields      []string
	CommandWord string
}

func Build(raw string) Command {
	raw = strings.TrimSpace(raw)
	fields := strings.Fields(raw)
	commandWord := commandWord(fields)
	return Command{
		Raw:         raw,
		Fields:      fields,
		CommandWord: commandWord,
	}
}

func Compile(cfg config.Config) []Compiled {
	compiled := make([]Compiled, 0, len(cfg.Rules.Rule))
	for _, rule := range cfg.Rules.Rule {
		fields := strings.Fields(rule.Pattern)
		entry := Compiled{
			Pattern:      rule.Pattern,
			Action:       strings.ToLower(rule.Action),
			Fields:       fields,
			OriginalRule: rule,
		}
		if strings.HasSuffix(rule.Pattern, "*") {
			entry.Kind = KindGlobPrefix
			entry.LastPrefix = strings.TrimSuffix(fields[len(fields)-1], "*")
		} else if len(fields) == 1 {
			entry.Kind = KindExactWord
		} else {
			entry.Kind = KindPrefixWords
		}
		compiled = append(compiled, entry)
	}
	return compiled
}

func Evaluate(cfg config.Config, raw string) Match {
	command := Build(raw)
	if command.Raw == "" {
		return Match{Action: "allow"}
	}

	for _, allowed := range cfg.Allowlist.Always {
		if command.CommandWord == allowed {
			return Match{
				AllowedByAllowlist: true,
				Action:             "allow",
				CommandWord:        command.CommandWord,
			}
		}
	}

	for _, rule := range Compile(cfg) {
		if rule.matches(command) {
			return Match{
				Matched:     true,
				Pattern:     rule.Pattern,
				Action:      rule.Action,
				Kind:        rule.Kind,
				CommandWord: command.CommandWord,
			}
		}
	}

	return Match{
		Action:      strings.ToLower(cfg.Rules.DefaultAction),
		CommandWord: command.CommandWord,
	}
}

func (c Compiled) matches(command Command) bool {
	switch c.Kind {
	case KindExactWord:
		return command.CommandWord == c.Fields[0]
	case KindPrefixWords:
		if len(command.Fields) < len(c.Fields) {
			return false
		}
		for i, field := range c.Fields {
			if command.Fields[i] != field {
				return false
			}
		}
		return true
	case KindGlobPrefix:
		if len(command.Fields) < len(c.Fields) {
			return false
		}
		for i := 0; i < len(c.Fields)-1; i++ {
			if command.Fields[i] != c.Fields[i] {
				return false
			}
		}
		lastIndex := len(c.Fields) - 1
		return strings.HasPrefix(command.Fields[lastIndex], c.LastPrefix)
	default:
		return false
	}
}

func commandWord(fields []string) string {
	for _, field := range fields {
		if !isAssignment(field) {
			return field
		}
	}
	return ""
}

func isAssignment(value string) bool {
	if value == "" {
		return false
	}
	index := strings.IndexByte(value, '=')
	if index <= 0 {
		return false
	}
	name := value[:index]
	for i, r := range name {
		if i == 0 {
			if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
				return false
			}
			continue
		}
		if !(r == '_' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
			return false
		}
	}
	return true
}
