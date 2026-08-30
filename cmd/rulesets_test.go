package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/nobuo-miura/gh-new-repo/internal/config"
	"github.com/nobuo-miura/gh-new-repo/internal/i18n"
	"github.com/nobuo-miura/gh-new-repo/internal/repo"
)

func TestRulesetLineRe(t *testing.T) {
	ok := map[string][2]string{
		"branch-main                    target=branch enforcement=active rules=4": {"branch-main", "target=branch enforcement=active rules=4"},
		"tag_release\ttarget=tag enforcement=active rules=5":                      {"tag_release", "target=tag enforcement=active rules=5"},
	}
	for line, want := range ok {
		m := rulesetLineRe.FindStringSubmatch(line)
		if m == nil {
			t.Errorf("no match: %q", line)
			continue
		}
		if m[1] != want[0] {
			t.Errorf("%q: name = %q, want %q", line, m[1], want[0])
		}
	}

	for _, line := range []string{
		"No rulesets found.",
		"",
		"Saved rulesets live in ~/.config/gh-rulekit/rulesets/",
	} {
		if rulesetLineRe.MatchString(line) {
			t.Errorf("should not match noise: %q", line)
		}
	}
}

// fakeGH は、rulekit サブコマンドが指定した終了コードを返す gh スタブを用意します。
func fakeGH(t *testing.T, rulekitExit int) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gh")
	script := "#!/bin/sh\nif [ \"$1\" = rulekit ]; then exit " + strconv.Itoa(rulekitExit) + "; fi\nexit 0\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GH_PATH", path)
}

func TestApplyRulesets_SkipsWhenRulekitUnavailable(t *testing.T) {
	fakeGH(t, 1) // 非0終了を、gh-rulekit を利用できない状態として扱います。

	if applyRulesets(i18n.For(i18n.EN), "acme", "svc", []string{"branch-main"}) {
		t.Error("applyRulesets must not report failure when gh-rulekit is unavailable")
	}
}

func TestApplyRulesets_NoNames(t *testing.T) {
	fakeGH(t, 1)
	if applyRulesets(i18n.For(i18n.EN), "acme", "svc", nil) {
		t.Error("no rulesets should never be a failure")
	}
}

func TestRenderSummary_ReflectsActualCreate(t *testing.T) {
	s := i18n.For(i18n.EN)
	p := repo.Plan{
		Name:        "svc",
		Description: "d",
		Visibility:  "private",
		Init:        config.Init{AddReadme: true, GitignoreTemplate: "Go", LicenseTemplate: "mit"},
	}

	plain := renderSummary(s, p, repo.Passthrough{}, "p")
	for _, want := range []string{"Description: d", "README", ".gitignore=Go", "license=mit"} {
		if !strings.Contains(plain, want) {
			t.Errorf("plain summary missing %q:\n%s", want, plain)
		}
	}

	src := renderSummary(s, p, repo.Passthrough{Source: "."}, "p")
	for _, unwanted := range []string{".gitignore=Go", "license=mit"} {
		if strings.Contains(src, unwanted) {
			t.Errorf("--source summary must not list %q:\n%s", unwanted, src)
		}
	}
	for _, want := range []string{"from local source .", "Description: d"} {
		if !strings.Contains(src, want) {
			t.Errorf("--source summary missing %q:\n%s", want, src)
		}
	}
}
