// Package repo は、プロファイルと入力値からリポジトリの作成計画を組み立てます。
//
// リポジトリの作成は公式の `gh repo create` へ委譲し、そこで指定できない Projects、
// Discussions、Pull Request のマージ設定などは、作成後に GitHub API で適用します。
package repo

import (
	"github.com/nobuo-miura/gh-new-repo/internal/config"
	"github.com/nobuo-miura/gh-new-repo/internal/ghapi"
)

// Plan は、1回のリポジトリ作成に使用する解決済みの設定です。
type Plan struct {
	Org         string // 空なら個人アカウント
	Name        string
	Description string
	Visibility  string // private | public | internal
	Init        config.Init
	Features    config.Features
	PR          config.PullRequests
	Rulesets    []string // gh-rulekit で適用するルールセット名
}

// FromProfile は、プロファイルから Plan の初期値を作成します。
func FromProfile(p config.Profile) Plan {
	return Plan{
		Description: p.Description,
		Visibility:  orDefault(p.Visibility, "private"),
		Init:        p.Init,
		Features:    p.Features,
		PR:          p.PullRequests,
		Rulesets:    append([]string(nil), p.Rulesets...),
	}
}

// Passthrough は、gh-new-repo で解釈せず `gh repo create` へ渡すフラグを保持します。
type Passthrough struct {
	Homepage           string
	Team               string
	Template           string
	Source             string
	Remote             string
	Clone              bool
	Push               bool
	IncludeAllBranches bool
}

// CreateArgs は、Plan と Passthrough から `gh repo create` の引数を組み立てます。
func (p Plan) CreateArgs(x Passthrough) []string {
	args := []string{"repo", "create"}

	name := p.Name
	if p.Org != "" && name != "" {
		name = p.Org + "/" + name
	}
	if name != "" {
		args = append(args, name)
	}

	switch p.Visibility {
	case "public":
		args = append(args, "--public")
	case "internal":
		args = append(args, "--internal")
	default:
		args = append(args, "--private")
	}

	if p.Description != "" {
		args = append(args, "--description", p.Description)
	}
	// テンプレートまたはローカルソースから作成する場合、内容は生成元から供給されるため、
	// プロファイルの --gitignore、--license、--add-readme は渡しません。
	if x.Template == "" && x.Source == "" {
		if p.Init.GitignoreTemplate != "" {
			args = append(args, "--gitignore", p.Init.GitignoreTemplate)
		}
		if p.Init.LicenseTemplate != "" {
			args = append(args, "--license", p.Init.LicenseTemplate)
		}
		if p.Init.AddReadme {
			args = append(args, "--add-readme")
		}
	}
	// `gh repo create` は Issues と Wiki の無効化フラグだけを提供しています。
	if p.Features.Issues != nil && !*p.Features.Issues {
		args = append(args, "--disable-issues")
	}
	if p.Features.Wiki != nil && !*p.Features.Wiki {
		args = append(args, "--disable-wiki")
	}

	if x.Homepage != "" {
		args = append(args, "--homepage", x.Homepage)
	}
	if x.Team != "" {
		args = append(args, "--team", x.Team)
	}
	if x.Template != "" {
		args = append(args, "--template", x.Template)
	}
	if x.IncludeAllBranches {
		args = append(args, "--include-all-branches")
	}
	if x.Source != "" {
		args = append(args, "--source", x.Source)
	}
	if x.Remote != "" {
		args = append(args, "--remote", x.Remote)
	}
	if x.Clone {
		args = append(args, "--clone")
	}
	if x.Push {
		args = append(args, "--push")
	}
	return args
}

// SettingsBody は、PATCH /repos/{owner}/{repo} へ送るリクエストボディを返します。
// nil の真偽値はプロファイルで未指定のため、リクエストへ含めません。
func (p Plan) SettingsBody() map[string]any {
	b := map[string]any{}
	putBool(b, "has_issues", p.Features.Issues)
	putBool(b, "has_projects", p.Features.Projects)
	putBool(b, "has_wiki", p.Features.Wiki)
	putBool(b, "has_discussions", p.Features.Discussions)

	putBool(b, "allow_merge_commit", p.PR.AllowMergeCommit)
	putBool(b, "allow_squash_merge", p.PR.AllowSquashMerge)
	putBool(b, "allow_rebase_merge", p.PR.AllowRebaseMerge)
	putBool(b, "allow_auto_merge", p.PR.AllowAutoMerge)
	putBool(b, "delete_branch_on_merge", p.PR.DeleteBranchOnMerge)
	putBool(b, "allow_update_branch", p.PR.AllowUpdateBranch)

	// 対応するマージ方式が無効な状態でコミット文言を送ると、GitHub API は 422 を返します。
	// マージ方式が明示的に false の場合だけ、関連するコミット文言を除外します。
	if notFalse(p.PR.AllowSquashMerge) {
		putStr(b, "squash_merge_commit_title", p.PR.SquashMergeCommitTitle)
		putStr(b, "squash_merge_commit_message", p.PR.SquashMergeCommitMessage)
	}
	if notFalse(p.PR.AllowMergeCommit) {
		putStr(b, "merge_commit_title", p.PR.MergeCommitTitle)
		putStr(b, "merge_commit_message", p.PR.MergeCommitMessage)
	}
	return b
}

// ApplyExisting は、既存リポジトリへ Plan の設定を適用します。
// 適用する項目がない場合は API を呼び出しません。
func ApplyExisting(cl *ghapi.Client, owner, name string, p Plan) error {
	body := p.SettingsBody()
	if len(body) == 0 {
		return nil
	}
	return cl.UpdateRepo(owner, name, body)
}

func putBool(m map[string]any, key string, v *bool) {
	if v != nil {
		m[key] = *v
	}
}

func putStr(m map[string]any, key, v string) {
	if v != "" {
		m[key] = v
	}
}

func notFalse(v *bool) bool { return v == nil || *v }

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
