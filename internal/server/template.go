package server

import (
	"fmt"
	"os"
	"strings"

	"thistlebbs/internal/ansi"

	"gopkg.in/yaml.v3"
)

// MenuTemplate holds a screen template loaded from a text file with a YAML
// header. The header defines metadata (locator, prompt, keybindings, errors)
// and the visual template below the "---" separator defines the screen layout.
// An optional "---post---" divider separates the page header from a per-post
// template used in loop-based views.
type MenuTemplate struct {
	Locator string    `yaml:"locator"`
	Rule    string    `yaml:"rule"`
	Prompt  string    `yaml:"prompt"`
	Error   string    `yaml:"error"`
	Bind    []Binding `yaml:"bind"`

	visual string // page header template (before ---post---)
	post   string // per-post template (after ---post---, if present)
}

// Binding maps one or more key strings to an action name.
type Binding struct {
	Keys   []string `yaml:"keys"`
	Action string   `yaml:"action"`
}

// LoadMenuTemplate reads a menu template file. The file format is:
//
//	# YAML header
//	locator: Front Gate
//	prompt: "\n> "
//	error: "'%s' is not a command."
//	bind:
//	  - keys: [l, login]
//	    action: login
//	---
//	# Visual template below uses {color} tags and {variable} tokens.
//	{blank}
//	{rule}
//	{cyan}  [L]ogin{reset}
func LoadMenuTemplate(path string) (*MenuTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load menu template %s: %w", path, err)
	}

	var t MenuTemplate
	parts := strings.SplitN(string(data), "---", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("menu template %s: missing --- separator", path)
	}

	header := parts[0]
	rest := parts[1]

	if err := yaml.Unmarshal([]byte(header), &t); err != nil {
		return nil, fmt.Errorf("menu template %s: yaml: %w", path, err)
	}

	// Split on ---post--- if present (for templates with per-post sections)
	postParts := strings.SplitN(rest, "---post---", 2)
	t.visual = postParts[0]
	if len(postParts) == 2 {
		t.post = postParts[1]
	}

	return &t, nil
}

// MatchInput checks the user's input against the keybindings and returns the
// matched action, or "" if no binding matches.
func (t *MenuTemplate) MatchInput(input string) string {
	lc := strings.ToLower(input)
	for _, b := range t.Bind {
		for _, k := range b.Keys {
			if lc == k {
				return b.Action
			}
		}
	}
	return ""
}

// Render expands {color} tags, {variable} tokens, {rule}, and {blank} in the
// visual template. Variables are supplied as a map.
func (t *MenuTemplate) Render(vars map[string]string, width int) string {
	s := t.visual

	// Expand {rule} → horizontal rule
	s = strings.ReplaceAll(s, "{rule}", ansi.Paint(ansi.BrightBlack, ansi.Rule(width, "")))

	// Expand {blank} → empty line
	s = strings.ReplaceAll(s, "{blank}", "")

	// Expand {variable} tokens
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}

	// Expand {color} tags
	s = ansi.ExpandTags(s)

	return s
}

// RenderLocator expands {variable} tokens in the locator string.
func (t *MenuTemplate) RenderLocator(vars map[string]string) string {
	s := t.Locator
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

// RenderRule expands {variable} tokens in the rule title. An empty result
// draws a plain unadorned rule.
func (t *MenuTemplate) RenderRule(vars map[string]string) string {
	s := t.Rule
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

// Errorf formats the error message template with the given choice.
func (t *MenuTemplate) Errorf(choice string) string {
	return fmt.Sprintf(t.Error, choice)
}

// HasPostTemplate reports whether this template has a post section (for loop rendering).
func (t *MenuTemplate) HasPostTemplate() bool {
	return t.post != ""
}

// RenderPost expands {color} tags, {variable} tokens, and {blank} in the
// post template section. Unlike Render, it does not expand {rule}.
func (t *MenuTemplate) RenderPost(vars map[string]string) string {
	s := t.post

	// Expand {blank} → empty line
	s = strings.ReplaceAll(s, "{blank}", "")

	// Expand {variable} tokens
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}

	// Expand {color} tags
	s = ansi.ExpandTags(s)

	return s
}
