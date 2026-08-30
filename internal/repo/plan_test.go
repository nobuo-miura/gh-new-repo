package repo

import (
	"slices"
	"strings"
	"testing"

	"github.com/nobuo-miura/gh-new-repo/internal/config"
)

func b(v bool) *bool { return &v }

func argValue(args []string, flag string) (string, bool) {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

func TestCreateArgs_Basics(t *testing.T) {
	p := Plan{Name: "my-repo", Org: "acme", Description: "hi", Visibility: "public"}
	args := p.CreateArgs(Passthrough{})

	if got := strings.Join(args[:2], " "); got != "repo create" {
		t.Fatalf("args must start with 'repo create', got %q", got)
	}
	if args[2] != "acme/my-repo" {
		t.Errorf("name arg = %q, want acme/my-repo", args[2])
	}
	if !slices.Contains(args, "--public") {
		t.Errorf("missing --public in %v", args)
	}
	if v, _ := argValue(args, "--description"); v != "hi" {
		t.Errorf("--description = %q", v)
	}
}

func TestCreateArgs_VisibilityDefault(t *testing.T) {
	for vis, want := range map[string]string{
		"":         "--private",
		"private":  "--private",
		"public":   "--public",
		"internal": "--internal",
	} {
		args := Plan{Name: "x", Visibility: vis}.CreateArgs(Passthrough{})
		if !slices.Contains(args, want) {
			t.Errorf("visibility %q: want %s in %v", vis, want, args)
		}
	}
}

func TestCreateArgs_FeaturesDisableOnly(t *testing.T) {
	// `gh repo create` は無効化フラグだけを提供するため、false の項目だけを渡します。
	args := Plan{Name: "x", Features: config.Features{Issues: b(false), Wiki: b(true)}}.CreateArgs(Passthrough{})
	if !slices.Contains(args, "--disable-issues") {
		t.Errorf("want --disable-issues in %v", args)
	}
	if slices.Contains(args, "--disable-wiki") {
		t.Errorf("wiki=true must not add --disable-wiki: %v", args)
	}
}

func TestCreateArgs_Passthrough(t *testing.T) {
	args := Plan{Name: "x"}.CreateArgs(Passthrough{
		Source: ".", Push: true, Clone: true, Template: "octo/tmpl", Homepage: "https://e.x",
	})
	for _, want := range []string{"--source", "--push", "--clone", "--template", "--homepage"} {
		if !slices.Contains(args, want) {
			t.Errorf("want %s in %v", want, args)
		}
	}
}

func TestCreateArgs_DropsContentFlagsForTemplateOrSource(t *testing.T) {
	p := Plan{Name: "x", Init: config.Init{AddReadme: true, LicenseTemplate: "mit", GitignoreTemplate: "Go"}}
	for _, x := range []Passthrough{{Template: "octo/tmpl"}, {Source: "."}} {
		args := p.CreateArgs(x)
		for _, unwanted := range []string{"--add-readme", "--license", "--gitignore"} {
			if slices.Contains(args, unwanted) {
				t.Errorf("%s must be dropped for %+v: %v", unwanted, x, args)
			}
		}
	}
	// 通常の作成では、プロファイルの初期化オプションを渡します。
	args := p.CreateArgs(Passthrough{})
	for _, want := range []string{"--add-readme", "--license", "--gitignore"} {
		if !slices.Contains(args, want) {
			t.Errorf("want %s in a plain create: %v", want, args)
		}
	}
}

func TestSettingsBody_SkipsNilBools(t *testing.T) {
	p := Plan{Features: config.Features{Issues: b(true)}}
	got := p.SettingsBody()
	if got["has_issues"] != true {
		t.Errorf("has_issues = %v, want true", got["has_issues"])
	}
	if _, ok := got["has_projects"]; ok {
		t.Error("has_projects should be omitted when nil")
	}
}

func TestSettingsBody_SquashFieldsGatedByMergeMethod(t *testing.T) {
	p := Plan{PR: config.PullRequests{AllowSquashMerge: b(false), SquashMergeCommitTitle: "PR_TITLE"}}
	if _, ok := p.SettingsBody()["squash_merge_commit_title"]; ok {
		t.Error("squash_merge_commit_title should be omitted when allow_squash_merge is false")
	}
	p.PR.AllowSquashMerge = nil
	if got := p.SettingsBody()["squash_merge_commit_title"]; got != "PR_TITLE" {
		t.Errorf("squash_merge_commit_title = %v, want PR_TITLE when allow_squash_merge is nil", got)
	}
}
