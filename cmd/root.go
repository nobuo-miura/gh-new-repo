// Package cmd は、gh-new-repo のコマンドラインインターフェースを提供します。
package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/nobuo-miura/gh-new-repo/internal/config"
	"github.com/nobuo-miura/gh-new-repo/internal/ghapi"
	"github.com/nobuo-miura/gh-new-repo/internal/i18n"
	"github.com/nobuo-miura/gh-new-repo/internal/repo"
	"github.com/nobuo-miura/gh-new-repo/internal/tui"
)

// createFlags は、gh-new-repo 固有のフラグと `gh repo create` 互換フラグを保持します。
type createFlags struct {
	// gh-new-repo 固有のフラグ
	profile     string
	lang        string
	interactive bool
	yes         bool
	dryRun      bool

	// Plan に反映する `gh repo create` 互換フラグ
	description   string
	gitignore     string
	license       string
	addReadme     bool
	disableIssues bool
	disableWiki   bool
	public        bool
	private       bool
	internal      bool

	// `gh repo create` へそのまま渡す互換フラグ
	homepage           string
	team               string
	template           string
	source             string
	remote             string
	clone              bool
	push               bool
	includeAllBranches bool
}

var flags createFlags

// Execute は、ルートコマンドを実行します。
func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "gh-new-repo [<name>]",
		Short: "Create a GitHub repository with your preferred settings",
		Long: "Create a GitHub repository with your preferred settings.\n\n" +
			"Repository creation is delegated to `gh repo create`, so its flags\n" +
			"(--public/--private/--internal, --clone, --source, --template, ...) work as-is.\n" +
			"Settings that `gh repo create` cannot set (Projects, Discussions, pull request\n" +
			"merge options, ...) are applied afterwards from the selected profile.\n\n" +
			"With no name, an interactive form opens. With a name, the default profile is\n" +
			"applied and you confirm before it is created.",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			return runNew(cmd, name)
		},
	}

	root.PersistentFlags().StringVar(&flags.lang, "lang", "", "UI language: en | ja (default: auto)")

	f := root.Flags()
	f.StringVar(&flags.profile, "profile", "", "profile to use (default: config default_profile)")
	f.BoolVarP(&flags.interactive, "interactive", "i", false, "open the full form even when a name is given")
	f.BoolVarP(&flags.yes, "yes", "y", false, "skip the confirmation prompt")
	f.BoolVar(&flags.dryRun, "dry-run", false, "print the gh repo create command and settings body, then exit")

	// `gh repo create` 互換フラグ
	f.StringVarP(&flags.description, "description", "d", "", "Description of the repository")
	f.StringVarP(&flags.gitignore, "gitignore", "g", "", "Specify a gitignore template for the repository")
	f.StringVarP(&flags.license, "license", "l", "", "Specify an Open Source License for the repository")
	f.BoolVar(&flags.addReadme, "add-readme", false, "Add a README file to the new repository")
	f.BoolVar(&flags.disableIssues, "disable-issues", false, "Disable issues in the new repository")
	f.BoolVar(&flags.disableWiki, "disable-wiki", false, "Disable wiki in the new repository")
	f.BoolVar(&flags.public, "public", false, "Make the new repository public")
	f.BoolVar(&flags.private, "private", false, "Make the new repository private")
	f.BoolVar(&flags.internal, "internal", false, "Make the new repository internal")
	f.StringVar(&flags.homepage, "homepage", "", "Repository home page URL")
	f.StringVarP(&flags.team, "team", "t", "", "The name of the organization team to be granted access")
	f.StringVarP(&flags.template, "template", "p", "", "Make the new repository based on a template repository")
	f.StringVarP(&flags.source, "source", "s", "", "Specify path to local repository to use as source")
	f.StringVarP(&flags.remote, "remote", "r", "", "Specify remote name for the new repository")
	f.BoolVarP(&flags.clone, "clone", "c", false, "Clone the new repository to the current directory")
	f.BoolVar(&flags.push, "push", false, "Push local commits to the new repository")
	f.BoolVar(&flags.includeAllBranches, "include-all-branches", false, "Include all branches from template repository")

	// 公式の `gh repo create` と同様に、可視性フラグは同時に指定できません。
	root.MarkFlagsMutuallyExclusive("public", "private", "internal")

	root.AddCommand(newProfileCmd(), newApplyCmd(), newConfigCmd())
	return root
}

func loadForLang() (*config.Config, i18n.Strings, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, i18n.Strings{}, err
	}
	lang := i18n.Detect(firstNonEmpty(flags.lang, cfg.Language))
	return cfg, i18n.For(lang), nil
}

func runNew(cmd *cobra.Command, name string) error {
	cfg, s, err := loadForLang()
	if err != nil {
		return err
	}
	if len(cfg.Profiles) == 0 {
		return errors.New(s.NoProfilesHint)
	}

	interactive := (name == "" && flags.source == "") || flags.interactive

	profileName := flags.profile
	if interactive && profileName == "" && len(cfg.Profiles) > 1 {
		picked, ok, err := tui.SelectProfile(s, cfg.Names(), cfg.DefaultProfile)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, s.Cancelled)
			return nil
		}
		profileName = picked
	}

	pname, profile, err := cfg.Resolve(profileName)
	if err != nil {
		if errors.Is(err, config.ErrProfileNotFound) {
			return fmt.Errorf(s.ProfileNotFound, firstNonEmpty(profileName, cfg.DefaultProfile))
		}
		return err
	}

	plan := repo.FromProfile(profile)
	plan.Name = name
	mergeFlags(cmd, &plan)
	pass := passthrough()

	var cl *ghapi.Client
	if interactive {
		if cl, err = ghapi.New(); err != nil {
			return err
		}
		meta, err := loadMeta(cl)
		if err != nil {
			return err
		}
		ok, err := tui.RunCreate(s, &plan, meta)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, s.Cancelled)
			return nil
		}
	} else {
		fmt.Fprint(os.Stderr, renderSummary(s, plan, pass, pname))
		if !flags.yes && !flags.dryRun {
			ok, err := confirmPrompt(s)
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(os.Stderr, s.Cancelled)
				return nil
			}
		}
	}

	if flags.dryRun {
		return printDryRun(plan, pass)
	}
	if cl == nil {
		if cl, err = ghapi.New(); err != nil {
			return err
		}
	}
	return execute(s, cl, plan, pass)
}

// mergeFlags は、明示された `gh repo create` 互換フラグを Plan へ反映します。
// 文字列フラグは、未指定と空文字の明示指定を区別するため Flag.Changed で判定します。
func mergeFlags(cmd *cobra.Command, p *repo.Plan) {
	changed := cmd.Flags().Changed

	if changed("description") {
		p.Description = flags.description
	}
	if changed("gitignore") {
		p.Init.GitignoreTemplate = flags.gitignore
	}
	if changed("license") {
		p.Init.LicenseTemplate = flags.license
	}
	if flags.addReadme {
		p.Init.AddReadme = true
	}
	if flags.disableIssues {
		p.Features.Issues = boolPtr(false)
	}
	if flags.disableWiki {
		p.Features.Wiki = boolPtr(false)
	}
	// 可視性フラグは MarkFlagsMutuallyExclusive で排他制御されているため、
	// この時点で true になるのは最大1つです。
	switch {
	case flags.public:
		p.Visibility = "public"
	case flags.private:
		p.Visibility = "private"
	case flags.internal:
		p.Visibility = "internal"
	}
}

func passthrough() repo.Passthrough {
	return repo.Passthrough{
		Homepage:           flags.homepage,
		Team:               flags.team,
		Template:           flags.template,
		Source:             flags.source,
		Remote:             flags.remote,
		Clone:              flags.clone,
		Push:               flags.push,
		IncludeAllBranches: flags.includeAllBranches,
	}
}

func loadMeta(cl *ghapi.Client) (tui.Meta, error) {
	orgs, err := cl.Orgs()
	if err != nil {
		return tui.Meta{}, err
	}
	gitignore, err := cl.GitignoreTemplates()
	if err != nil {
		return tui.Meta{}, err
	}
	licenses, err := cl.Licenses()
	if err != nil {
		return tui.Meta{}, err
	}
	return tui.Meta{
		Orgs:      orgs,
		Gitignore: gitignore,
		Licenses:  licenses,
		Rulesets:  listRulesets(), // gh-rulekit 未インストールなら空
	}, nil
}

// `gh rulekit list` は、1件を次の形式で出力します。
//
//	<name>   target=branch enforcement=active rules=4
var rulesetLineRe = regexp.MustCompile(`^([A-Za-z0-9._-]+)\s+(target=\S.*)$`)

// listRulesets は、`gh rulekit list` の出力から選択肢を取得します。
// gh-rulekit を利用できない場合や保存済みルールセットがない場合は nil を返します。
func listRulesets() []tui.RulesetOption {
	out, _, err := runGHCapture("rulekit", "list")
	if err != nil {
		return nil
	}
	var opts []tui.RulesetOption
	for _, line := range strings.Split(out, "\n") {
		m := rulesetLineRe.FindStringSubmatch(strings.TrimRight(line, " \t"))
		if m == nil {
			continue
		}
		opts = append(opts, tui.RulesetOption{Name: m[1], Summary: strings.TrimSpace(m[2])})
	}
	return opts
}

func confirmPrompt(s i18n.Strings) (bool, error) {
	ok := true
	err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(s.ConfirmCreate).Value(&ok).Affirmative(s.Yes).Negative(s.No),
	)).Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return false, nil
	}
	return ok, err
}

func execute(s i18n.Strings, cl *ghapi.Client, plan repo.Plan, pass repo.Passthrough) error {
	args := plan.CreateArgs(pass)

	fmt.Fprintf(os.Stderr, "→ %s: gh %s\n", s.StepCreatingRepo, strings.Join(args, " "))
	out, err := runGH(args)
	if err != nil {
		return err
	}

	owner, name, err := resolveCreated(out, plan, pass, cl)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "✓ %s: %s/%s\n", s.StepRepoCreated, owner, name)

	failed := false
	if body := plan.SettingsBody(); len(body) > 0 {
		if err := cl.UpdateRepo(owner, name, body); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s %s: %v\n", s.StepApplyingConfig, s.StepFailed, err)
			failed = true
		} else {
			fmt.Fprintf(os.Stderr, "✓ %s\n", s.StepConfigApplied)
		}
	}

	if applyRulesets(s, owner, name, plan.Rulesets) {
		failed = true
	}

	if !failed {
		fmt.Fprintf(os.Stderr, "✓ %s\n", s.AllDone)
	}

	if url := firstRepoURL(out); url != "" {
		fmt.Println(url)
	} else {
		fmt.Printf("https://github.com/%s/%s\n", owner, name)
	}
	if failed {
		fmt.Fprintf(os.Stderr, s.PartialFailure+"\n", owner+"/"+name, owner+"/"+name)
		return errSilent
	}
	return nil
}

// errSilent は、エラーを表示済みの処理から追加メッセージなしで失敗を返すために使います。
var errSilent = errors.New("")

// rulekitAvailable は、gh-rulekit 拡張機能を利用できるかどうかを返します。
func rulekitAvailable() bool {
	_, _, err := runGHCapture("rulekit", "list")
	return err == nil
}

// applyRulesets は、指定されたルールセットを `gh rulekit apply` で適用します。
// gh-rulekit を利用できない場合は失敗とせずにスキップし、適用に1件でも失敗すると
// true を返します。
func applyRulesets(s i18n.Strings, owner, name string, names []string) (failed bool) {
	if len(names) == 0 {
		return false
	}
	if !rulekitAvailable() {
		fmt.Fprintf(os.Stderr, "! "+s.RulekitNotInstalled+"\n", len(names))
		return false
	}
	target := owner + "/" + name
	for _, rs := range names {
		fmt.Fprintf(os.Stderr, "→ "+s.StepApplyingRuleset+"\n", rs)
		if _, _, err := runGHCapture("rulekit", "apply", rs, "--repo", target); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s %s: %v\n", fmt.Sprintf(s.StepApplyingRuleset, rs), s.StepFailed, err)
			failed = true
		} else {
			fmt.Fprintf(os.Stderr, "✓ "+s.RulesetApplied+"\n", rs)
		}
	}
	return failed
}

// runGH は `gh <args>` を実行し、標準エラー出力を転送して標準出力を返します。
func runGH(args []string) (string, error) {
	bin := firstNonEmpty(os.Getenv("GH_PATH"), "gh")
	cmd := exec.Command(bin, args...) //nolint:gosec // args は内部で組み立て済み
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh %s: %w", strings.Join(args, " "), err)
	}
	return stdout.String(), nil
}

// runGHCapture は `gh <args>` を実行し、標準出力と標準エラー出力を取り込みます。
func runGHCapture(args ...string) (stdout, stderr string, err error) {
	bin := firstNonEmpty(os.Getenv("GH_PATH"), "gh")
	cmd := exec.Command(bin, args...) //nolint:gosec // args は内部で組み立て済み
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	if runErr := cmd.Run(); runErr != nil {
		msg := strings.TrimSpace(se.String())
		if msg == "" {
			msg = runErr.Error()
		}
		return so.String(), se.String(), errors.New(msg)
	}
	return so.String(), se.String(), nil
}

var repoURLRe = regexp.MustCompile(`https?://[^/\s]*github\.[^/\s]+/([^/\s]+)/([^/\s]+?)(?:\.git)?/?$`)

func firstRepoURL(out string) string {
	for _, line := range strings.Fields(out) {
		if repoURLRe.MatchString(line) {
			return line
		}
	}
	return ""
}

// resolveCreated は、`gh repo create` の出力から作成された owner/repo を特定します。
// 出力から特定できない場合は、Plan と現在のログインユーザーから補完します。
func resolveCreated(out string, plan repo.Plan, pass repo.Passthrough, cl *ghapi.Client) (owner, name string, err error) {
	for _, line := range strings.Fields(out) {
		if m := repoURLRe.FindStringSubmatch(line); m != nil {
			return m[1], m[2], nil
		}
	}

	name = plan.Name
	if name == "" && pass.Source != "" {
		abs, e := filepath.Abs(pass.Source)
		if e != nil {
			return "", "", e
		}
		name = filepath.Base(abs)
	}
	owner = plan.Org
	if o, n, ok := strings.Cut(name, "/"); ok {
		owner, name = o, n
	}
	if owner == "" {
		if owner, err = cl.Login(); err != nil {
			return "", "", err
		}
	}
	if name == "" {
		return "", "", errors.New("could not determine the created repository name")
	}
	return owner, name, nil
}

func printDryRun(plan repo.Plan, pass repo.Passthrough) error {
	fmt.Printf("gh %s\n", strings.Join(plan.CreateArgs(pass), " "))
	if body := plan.SettingsBody(); len(body) > 0 {
		fmt.Println("PATCH /repos/{owner}/{repo}")
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(body); err != nil {
			return err
		}
	}
	for _, rs := range plan.Rulesets {
		fmt.Printf("gh rulekit apply %s --repo {owner}/{repo}\n", rs)
	}
	return nil
}

func renderSummary(s i18n.Strings, plan repo.Plan, pass repo.Passthrough, profileName string) string {
	name := firstNonEmpty(plan.Name, pass.Source)
	owner := plan.Org
	if o, n, ok := strings.Cut(name, "/"); ok {
		owner, name = o, n
	}
	ownerLabel := s.OwnerPersonal
	if owner != "" {
		ownerLabel = owner
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", s.SummaryHeader)
	fmt.Fprintf(&b, "  %s: %s\n", s.RepoNameTitle, name)
	fmt.Fprintf(&b, "  %s: %s\n", s.ProfileLabel, profileName)
	fmt.Fprintf(&b, "  %s: %s\n", s.OwnerTitle, ownerLabel)
	fmt.Fprintf(&b, "  %s: %s\n", s.VisibilityTitle, visibilityLabel(s, plan.Visibility))
	if plan.Description != "" {
		fmt.Fprintf(&b, "  %s: %s\n", s.DescriptionTitle, plan.Description)
	}

	// 実際に `gh repo create` へ渡す初期化方法だけを表示します。
	// --source または --template の使用時は、プロファイルの初期化設定を適用しません。
	switch {
	case pass.Template != "":
		fmt.Fprintf(&b, "  %s: "+s.SummaryFromTemplate+"\n", s.InitGroupTitle, pass.Template)
	case pass.Source != "":
		fmt.Fprintf(&b, "  %s: "+s.SummaryFromSource+"\n", s.InitGroupTitle, pass.Source)
	default:
		init := []string{}
		if plan.Init.AddReadme {
			init = append(init, "README")
		}
		if plan.Init.GitignoreTemplate != "" {
			init = append(init, ".gitignore="+plan.Init.GitignoreTemplate)
		}
		if plan.Init.LicenseTemplate != "" {
			init = append(init, "license="+plan.Init.LicenseTemplate)
		}
		if len(init) > 0 {
			fmt.Fprintf(&b, "  %s: %s\n", s.InitGroupTitle, strings.Join(init, ", "))
		}
	}

	fmt.Fprintf(&b, "  %s: %s=%s %s=%s %s=%s %s=%s\n", s.FeaturesGroupTitle,
		s.FeatIssues, triState(s, plan.Features.Issues),
		s.FeatProjects, triState(s, plan.Features.Projects),
		s.FeatWiki, triState(s, plan.Features.Wiki),
		s.FeatDiscussions, triState(s, plan.Features.Discussions))
	fmt.Fprintf(&b, "  %s: merge=%s squash=%s rebase=%s / %s=%s\n", s.PRGroupTitle,
		triState(s, plan.PR.AllowMergeCommit),
		triState(s, plan.PR.AllowSquashMerge),
		triState(s, plan.PR.AllowRebaseMerge),
		s.PRDeleteBranchOnMerge, triState(s, plan.PR.DeleteBranchOnMerge))
	if len(plan.Rulesets) > 0 {
		fmt.Fprintf(&b, "  %s: %s\n", s.RulesetsLabel, strings.Join(plan.Rulesets, ", "))
	}
	return b.String()
}

func visibilityLabel(s i18n.Strings, v string) string {
	switch v {
	case "public":
		return s.VisibilityPublic
	case "internal":
		return s.VisibilityInternal
	default:
		return s.VisibilityPrivate
	}
}

func triState(s i18n.Strings, v *bool) string {
	switch {
	case v == nil:
		return "-"
	case *v:
		return s.Yes
	default:
		return s.No
	}
}

func boolPtr(b bool) *bool { return &b }

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
