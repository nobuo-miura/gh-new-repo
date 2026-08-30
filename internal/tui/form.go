// Package tui は、huh を使用したリポジトリ作成フォームとプロファイル編集フォームを提供します。
package tui

import (
	"errors"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/nobuo-miura/gh-new-repo/internal/config"
	"github.com/nobuo-miura/gh-new-repo/internal/ghapi"
	"github.com/nobuo-miura/gh-new-repo/internal/i18n"
	"github.com/nobuo-miura/gh-new-repo/internal/repo"
)

// RulesetOption は、gh-rulekit に保存されたルールセットの選択肢です。
type RulesetOption struct {
	Name    string
	Summary string // 一覧表示用の補足（target / enforcement / rules 数など）
}

// Meta は、フォームの選択肢に使用する外部データを保持します。
type Meta struct {
	Orgs      []string // 所属 Organization。個人アカウントは別途先頭に加える
	Gitignore []string // .gitignore テンプレート名
	Licenses  []ghapi.License
	Rulesets  []RulesetOption // gh-rulekit が未インストールなら空
}

var repoNameRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// GitHub API が受け付ける squash/merge コミット文言の選択肢です。
var (
	squashTitleValues   = []string{"PR_TITLE", "COMMIT_OR_PR_TITLE"}
	squashMessageValues = []string{"PR_BODY", "COMMIT_MESSAGES", "BLANK"}
	mergeTitleValues    = []string{"MERGE_MESSAGE", "PR_TITLE"}
	mergeMessageValues  = []string{"PR_BODY", "PR_TITLE", "BLANK"}
)

// fields は、フォームで編集する値を保持します。*bool は bool へ展開して扱います。
type fields struct {
	visibility                            string
	addReadme                             bool
	gitignore, license                    string
	issues, projects, wiki, discussions   bool
	allowMerge, allowSquash, allowRebase  bool
	allowAuto, deleteBranch, updateBranch bool
	squashTitle, squashMessage            string
	mergeTitle, mergeMessage              string
	rulesets                              []string
}

func fieldsFromProfile(p config.Profile) fields {
	return fields{
		rulesets:      append([]string(nil), p.Rulesets...),
		visibility:    orDefault(p.Visibility, "private"),
		addReadme:     p.Init.AddReadme,
		gitignore:     p.Init.GitignoreTemplate,
		license:       p.Init.LicenseTemplate,
		issues:        deref(p.Features.Issues),
		projects:      deref(p.Features.Projects),
		wiki:          deref(p.Features.Wiki),
		discussions:   deref(p.Features.Discussions),
		allowMerge:    deref(p.PullRequests.AllowMergeCommit),
		allowSquash:   derefTrue(p.PullRequests.AllowSquashMerge),
		allowRebase:   deref(p.PullRequests.AllowRebaseMerge),
		allowAuto:     deref(p.PullRequests.AllowAutoMerge),
		deleteBranch:  deref(p.PullRequests.DeleteBranchOnMerge),
		updateBranch:  deref(p.PullRequests.AllowUpdateBranch),
		squashTitle:   orDefault(p.PullRequests.SquashMergeCommitTitle, "PR_TITLE"),
		squashMessage: orDefault(p.PullRequests.SquashMergeCommitMessage, "PR_BODY"),
		mergeTitle:    orDefault(p.PullRequests.MergeCommitTitle, "MERGE_MESSAGE"),
		mergeMessage:  orDefault(p.PullRequests.MergeCommitMessage, "PR_BODY"),
	}
}

// toProfile は、フォームの編集結果を Profile へ変換します。
// フォームに表示した真偽値はすべて明示的に設定します。
func (f fields) toProfile(base config.Profile) config.Profile {
	base.Visibility = f.visibility
	base.Init = config.Init{
		AddReadme:         f.addReadme,
		GitignoreTemplate: f.gitignore,
		LicenseTemplate:   f.license,
	}
	base.Features = config.Features{
		Issues: boolp(f.issues), Projects: boolp(f.projects),
		Wiki: boolp(f.wiki), Discussions: boolp(f.discussions),
	}
	base.PullRequests = config.PullRequests{
		AllowMergeCommit:         boolp(f.allowMerge),
		AllowSquashMerge:         boolp(f.allowSquash),
		AllowRebaseMerge:         boolp(f.allowRebase),
		AllowAutoMerge:           boolp(f.allowAuto),
		DeleteBranchOnMerge:      boolp(f.deleteBranch),
		AllowUpdateBranch:        boolp(f.updateBranch),
		SquashMergeCommitTitle:   f.squashTitle,
		SquashMergeCommitMessage: f.squashMessage,
		MergeCommitTitle:         f.mergeTitle,
		MergeCommitMessage:       f.mergeMessage,
	}
	base.Rulesets = append([]string(nil), f.rulesets...)
	return base
}

// settingsGroups は、作成フォームとプロファイル編集フォームで共用する設定グループを返します。
func settingsGroups(s i18n.Strings, f *fields, meta Meta) []*huh.Group {
	gitignoreOpts := []huh.Option[string]{huh.NewOption(s.GitignoreNone, "")}
	for _, g := range meta.Gitignore {
		gitignoreOpts = append(gitignoreOpts, huh.NewOption(g, g))
	}
	licenseOpts := []huh.Option[string]{huh.NewOption(s.LicenseNone, "")}
	for _, l := range meta.Licenses {
		licenseOpts = append(licenseOpts, huh.NewOption(l.Name, l.Key))
	}

	groups := []*huh.Group{
		huh.NewGroup(
			huh.NewConfirm().Title(s.AddReadmeTitle).Value(&f.addReadme).
				Affirmative(s.Yes).Negative(s.No),
			huh.NewSelect[string]().Title(s.GitignoreTitle).Description(s.GitignoreHelp).
				Options(gitignoreOpts...).Value(&f.gitignore).Height(12).Filtering(true),
			huh.NewSelect[string]().Title(s.LicenseTitle).Description(s.LicenseHelp).
				Options(licenseOpts...).Value(&f.license).Height(12).Filtering(true),
		).Title(s.InitGroupTitle),

		huh.NewGroup(
			confirm(s, s.FeatIssues, &f.issues),
			confirm(s, s.FeatProjects, &f.projects),
			confirm(s, s.FeatWiki, &f.wiki),
			confirm(s, s.FeatDiscussions, &f.discussions),
		).Title(s.FeaturesGroupTitle),

		huh.NewGroup(
			confirm(s, s.PRAllowMergeCommit, &f.allowMerge),
			confirm(s, s.PRAllowSquashMerge, &f.allowSquash),
			confirm(s, s.PRAllowRebaseMerge, &f.allowRebase),
			confirm(s, s.PRAllowAutoMerge, &f.allowAuto),
			confirm(s, s.PRDeleteBranchOnMerge, &f.deleteBranch),
			confirm(s, s.PRAllowUpdateBranch, &f.updateBranch),
			huh.NewNote().Description(s.PRCommitTitleNote),
			huh.NewSelect[string]().Title(s.PRSquashTitleTitle).
				Options(huh.NewOptions(squashTitleValues...)...).Value(&f.squashTitle),
			huh.NewSelect[string]().Title(s.PRSquashMessageTitle).
				Options(huh.NewOptions(squashMessageValues...)...).Value(&f.squashMessage),
			huh.NewSelect[string]().Title(s.PRMergeTitleTitle).
				Options(huh.NewOptions(mergeTitleValues...)...).Value(&f.mergeTitle),
			huh.NewSelect[string]().Title(s.PRMergeMessageTitle).
				Options(huh.NewOptions(mergeMessageValues...)...).Value(&f.mergeMessage),
		).Title(s.PRGroupTitle),
	}

	if len(meta.Rulesets) > 0 {
		selected := map[string]bool{}
		for _, name := range f.rulesets {
			selected[name] = true
		}
		opts := make([]huh.Option[string], 0, len(meta.Rulesets))
		for _, rs := range meta.Rulesets {
			label := rs.Name
			if rs.Summary != "" {
				label = rs.Name + "  (" + rs.Summary + ")"
			}
			opts = append(opts, huh.NewOption(label, rs.Name).Selected(selected[rs.Name]))
		}
		groups = append(groups, huh.NewGroup(
			huh.NewMultiSelect[string]().Title(s.RulesetsGroupTitle).Description(s.RulesetsHelp).
				Options(opts...).Value(&f.rulesets).Height(10),
		).Title(s.RulesetsGroupTitle))
	}

	return groups
}

func visibilityOptions(s i18n.Strings) []huh.Option[string] {
	return []huh.Option[string]{
		huh.NewOption(s.VisibilityPrivate, "private"),
		huh.NewOption(s.VisibilityPublic, "public"),
		huh.NewOption(s.VisibilityInternal, "internal"),
	}
}

// SelectProfile は、プロファイルを選択するフォームを表示します。
func SelectProfile(s i18n.Strings, names []string, def string) (name string, ok bool, err error) {
	selected := def
	opts := make([]huh.Option[string], 0, len(names))
	for _, n := range names {
		opts = append(opts, huh.NewOption(n, n))
	}
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title(s.SelectProfileTitle).Description(s.SelectProfileHelp).
			Options(opts...).Value(&selected),
	))
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", false, nil
		}
		return "", false, err
	}
	return selected, true, nil
}

// RunCreate は、リポジトリ作成フォームを表示します。
// 確定時は plan を更新して true を返し、中断時は plan を変更せず false を返します。
func RunCreate(s i18n.Strings, plan *repo.Plan, meta Meta) (confirmed bool, err error) {
	f := fieldsFromProfile(planToProfile(*plan))
	name := plan.Name
	desc := plan.Description
	owner := plan.Org
	proceed := true

	ownerOpts := []huh.Option[string]{huh.NewOption(s.OwnerPersonal, "")}
	for _, o := range meta.Orgs {
		ownerOpts = append(ownerOpts, huh.NewOption(o, o))
	}

	groups := []*huh.Group{
		huh.NewGroup(
			huh.NewInput().Title(s.RepoNameTitle).Description(s.RepoNameHelp).
				Value(&name).Validate(validateRepoName(s)),
			huh.NewInput().Title(s.DescriptionTitle).Value(&desc),
			huh.NewSelect[string]().Title(s.OwnerTitle).Description(s.OwnerHelp).
				Options(ownerOpts...).Value(&owner),
			huh.NewSelect[string]().Title(s.VisibilityTitle).
				Options(visibilityOptions(s)...).Value(&f.visibility),
		).Title(s.GeneralGroupTitle),
	}
	groups = append(groups, settingsGroups(s, &f, meta)...)
	groups = append(groups, huh.NewGroup(
		huh.NewConfirm().Title(s.ConfirmCreate).Value(&proceed).
			Affirmative(s.Yes).Negative(s.No),
	).Title(s.ConfirmGroupTitle))

	if err := huh.NewForm(groups...).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, nil
		}
		return false, err
	}
	if !proceed {
		return false, nil
	}

	p := f.toProfile(config.Profile{Description: strings.TrimSpace(desc)})
	*plan = repo.FromProfile(p)
	plan.Name = strings.TrimSpace(name)
	plan.Org = owner
	return true, nil
}

// RunProfileEdit は、プロファイルの設定項目を編集するフォームを表示します。
func RunProfileEdit(s i18n.Strings, p *config.Profile, meta Meta) (saved bool, err error) {
	f := fieldsFromProfile(*p)
	proceed := true

	groups := []*huh.Group{
		huh.NewGroup(
			huh.NewSelect[string]().Title(s.VisibilityTitle).
				Options(visibilityOptions(s)...).Value(&f.visibility),
		).Title(s.ProfileSettingsTitle),
	}
	groups = append(groups, settingsGroups(s, &f, meta)...)
	groups = append(groups, huh.NewGroup(
		huh.NewConfirm().Title(s.SaveProfileConfirm).Value(&proceed).
			Affirmative(s.Yes).Negative(s.No),
	).Title(s.ConfirmGroupTitle))

	if err := huh.NewForm(groups...).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, nil
		}
		return false, err
	}
	if !proceed {
		return false, nil
	}
	*p = f.toProfile(*p)
	return true, nil
}

func validateRepoName(s i18n.Strings) func(string) error {
	return func(v string) error {
		v = strings.TrimSpace(v)
		if v == "" {
			return errors.New(s.RepoNameRequiredErr)
		}
		if !repoNameRe.MatchString(v) {
			return errors.New(s.RepoNameInvalidErr)
		}
		return nil
	}
}

func confirm(s i18n.Strings, title string, v *bool) *huh.Confirm {
	return huh.NewConfirm().Title(title).Value(v).Affirmative(s.Yes).Negative(s.No)
}

// planToProfile は、フォームの初期値に使用するため Plan を Profile へ変換します。
func planToProfile(p repo.Plan) config.Profile {
	return config.Profile{
		Description:  p.Description,
		Visibility:   p.Visibility,
		Init:         p.Init,
		Features:     p.Features,
		PullRequests: p.PR,
		Rulesets:     p.Rulesets,
	}
}

func deref(v *bool) bool     { return v != nil && *v }
func derefTrue(v *bool) bool { return v == nil || *v }
func boolp(b bool) *bool     { return &b }

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
