package server

import "testing"

func TestRenderRuleStatic(t *testing.T) {
	tmpl := &MenuTemplate{Rule: "Thistle BBS"}
	if got := tmpl.RenderRule(nil); got != "Thistle BBS" {
		t.Fatalf("expected %q, got %q", "Thistle BBS", got)
	}
}

func TestRenderRuleVars(t *testing.T) {
	tmpl := &MenuTemplate{Rule: "{title} [archive]"}
	got := tmpl.RenderRule(map[string]string{"title": "News"})
	if got != "News [archive]" {
		t.Fatalf("expected %q, got %q", "News [archive]", got)
	}
}

func TestRenderRuleEmpty(t *testing.T) {
	tmpl := &MenuTemplate{}
	if got := tmpl.RenderRule(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestRenderRuleUnknownTokenPreserved(t *testing.T) {
	tmpl := &MenuTemplate{Rule: "{unset}"}
	if got := tmpl.RenderRule(nil); got != "{unset}" {
		t.Fatalf("expected %q, got %q", "{unset}", got)
	}
}