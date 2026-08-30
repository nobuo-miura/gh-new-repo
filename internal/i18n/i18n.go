// Package i18n は、TUI とメッセージ出力の英語・日本語対応を提供します。
//
// 言語ごとに Strings 構造体を定義し、未定義の文言をコンパイル時に検出できるようにします。
package i18n

import (
	"os"
	"strings"
)

// Lang は、対応する表示言語を表します。
type Lang string

const (
	// EN は英語を表します。
	EN Lang = "en"
	// JA は日本語を表します。
	JA Lang = "ja"
)

// Strings は、UI に表示する固定文言を保持します。
type Strings struct {
	// 共通
	Yes string
	No  string

	// プロファイル選択
	SelectProfileTitle string
	SelectProfileHelp  string

	// General グループ
	GeneralGroupTitle   string
	RepoNameTitle       string
	RepoNameHelp        string
	RepoNameRequiredErr string
	RepoNameInvalidErr  string
	DescriptionTitle    string
	OwnerTitle          string
	OwnerHelp           string
	OwnerPersonal       string
	VisibilityTitle     string
	VisibilityPrivate   string
	VisibilityPublic    string
	VisibilityInternal  string

	// Init グループ
	InitGroupTitle string
	AddReadmeTitle string
	GitignoreTitle string
	GitignoreHelp  string
	GitignoreNone  string
	LicenseTitle   string
	LicenseHelp    string
	LicenseNone    string

	// Features グループ
	FeaturesGroupTitle string
	FeatIssues         string
	FeatProjects       string
	FeatWiki           string
	FeatDiscussions    string

	// Rulesets グループ（gh-rulekit 連携）
	RulesetsGroupTitle  string
	RulesetsHelp        string
	RulesetsLabel       string
	StepApplyingRuleset string // %s にルールセット名
	RulesetApplied      string // %s にルールセット名
	RulekitNotInstalled string // %d にスキップしたルールセット数

	// Pull Requests グループ
	PRGroupTitle          string
	PRAllowMergeCommit    string
	PRAllowSquashMerge    string
	PRAllowRebaseMerge    string
	PRAllowAutoMerge      string
	PRDeleteBranchOnMerge string
	PRAllowUpdateBranch   string
	PRSquashTitleTitle    string
	PRSquashMessageTitle  string
	PRMergeTitleTitle     string
	PRMergeMessageTitle   string
	PRCommitTitleNote     string

	// 確認・実行
	ConfirmGroupTitle   string
	ConfirmCreate       string
	SummaryHeader       string
	Cancelled           string
	ProfileLabel        string // サマリ内の「プロファイル」ラベル
	SummaryFromTemplate string // %s にテンプレートリポジトリ
	SummaryFromSource   string // %s にローカルソースのパス

	StepCreatingRepo   string
	StepRepoCreated    string
	StepApplyingConfig string
	StepConfigApplied  string
	StepFailed         string
	AllDone            string
	PartialFailure     string // %s にリポジトリのフルネームが入る

	// profile サブコマンド
	ProfileNotFound      string // %s
	NoEditorEnv          string
	ProfileSettingsTitle string
	SaveProfileConfirm   string
	ProfileCreated       string // %s
	ProfileUpdated       string // %s
	ProfileExists        string // %s
	ProfileFromNotFound  string // %s
	DefaultProfileSet    string // %s
	NoProfilesHint       string

	// config サブコマンド
	LanguageSettingTitle string
	LanguageSettingHelp  string
	LangAuto             string
	LangEnglish          string
	LangJapanese         string
	LanguageSet          string // %s
	InvalidLanguage      string
}

var (
	en = Strings{
		Yes: "Yes",
		No:  "No",

		SelectProfileTitle: "Profile",
		SelectProfileHelp:  "Preset of initial values to start from",

		GeneralGroupTitle:   "General",
		RepoNameTitle:       "Repository name",
		RepoNameHelp:        "The name of the new repository",
		RepoNameRequiredErr: "repository name is required",
		RepoNameInvalidErr:  "use only letters, numbers, hyphen, underscore and dot",
		DescriptionTitle:    "Description",
		OwnerTitle:          "Owner",
		OwnerHelp:           "Personal account or an organization",
		OwnerPersonal:       "(personal account)",
		VisibilityTitle:     "Visibility",
		VisibilityPrivate:   "Private",
		VisibilityPublic:    "Public",
		VisibilityInternal:  "Internal",

		InitGroupTitle: "Initialize",
		AddReadmeTitle: "Add a README file",
		GitignoreTitle: "Add .gitignore",
		GitignoreHelp:  "Choose a .gitignore template",
		GitignoreNone:  "None",
		LicenseTitle:   "Choose a license",
		LicenseHelp:    "A license tells others what they can and can't do with your code",
		LicenseNone:    "None",

		FeaturesGroupTitle: "Features",
		FeatIssues:         "Issues",
		FeatProjects:       "Projects",
		FeatWiki:           "Wikis",
		FeatDiscussions:    "Discussions",

		RulesetsGroupTitle:  "Rulesets",
		RulesetsHelp:        "Saved gh-rulekit rulesets to apply after the repository is created",
		RulesetsLabel:       "Rulesets",
		StepApplyingRuleset: "Applying ruleset %s",
		RulesetApplied:      "Ruleset %s applied",
		RulekitNotInstalled: "gh-rulekit is not installed; skipping %d ruleset(s)",

		PRGroupTitle:          "Pull Requests",
		PRAllowMergeCommit:    "Allow merge commits",
		PRAllowSquashMerge:    "Allow squash merging",
		PRAllowRebaseMerge:    "Allow rebase merging",
		PRAllowAutoMerge:      "Allow auto-merge",
		PRDeleteBranchOnMerge: "Automatically delete head branches",
		PRAllowUpdateBranch:   "Always suggest updating pull request branches",
		PRSquashTitleTitle:    "Default squash commit title",
		PRSquashMessageTitle:  "Default squash commit message",
		PRMergeTitleTitle:     "Default merge commit title",
		PRMergeMessageTitle:   "Default merge commit message",
		PRCommitTitleNote:     "Applied only when the corresponding merge method is enabled",

		ConfirmGroupTitle:   "Confirm",
		ConfirmCreate:       "Create this repository?",
		SummaryHeader:       "The following repository will be created:",
		Cancelled:           "Cancelled.",
		ProfileLabel:        "Profile",
		SummaryFromTemplate: "from template %s",
		SummaryFromSource:   "from local source %s",

		StepCreatingRepo:   "Creating repository",
		StepRepoCreated:    "Repository created",
		StepApplyingConfig: "Applying settings",
		StepConfigApplied:  "Settings applied",
		StepFailed:         "failed",
		AllDone:            "Done",
		PartialFailure:     "The repository %s was created, but some settings failed. Re-run: gh new-repo apply %s",

		ProfileNotFound:      "profile %q not found in config",
		NoEditorEnv:          "set $EDITOR or $VISUAL to edit the config",
		ProfileSettingsTitle: "Profile settings",
		SaveProfileConfirm:   "Save this profile?",
		ProfileCreated:       "Created profile %q",
		ProfileUpdated:       "Updated profile %q",
		ProfileExists:        "profile %q already exists (use `gh new-repo profile edit`)",
		ProfileFromNotFound:  "source profile %q not found",
		DefaultProfileSet:    "Default profile set to %q",
		NoProfilesHint:       "no profiles defined; create one with `gh new-repo profile new <name>`",

		LanguageSettingTitle: "Language",
		LanguageSettingHelp:  "Language for gh-new-repo's own prompts and messages",
		LangAuto:             "Auto (detect from LANG)",
		LangEnglish:          "English",
		LangJapanese:         "日本語",
		LanguageSet:          "Language set to %s",
		InvalidLanguage:      "language must be one of: auto, en, ja",
	}

	ja = Strings{
		Yes: "はい",
		No:  "いいえ",

		SelectProfileTitle: "プロファイル",
		SelectProfileHelp:  "初期値のプリセット",

		GeneralGroupTitle:   "基本",
		RepoNameTitle:       "リポジトリ名",
		RepoNameHelp:        "作成するリポジトリの名前",
		RepoNameRequiredErr: "リポジトリ名は必須です",
		RepoNameInvalidErr:  "英数字・ハイフン・アンダースコア・ドットのみ使用できます",
		DescriptionTitle:    "説明",
		OwnerTitle:          "オーナー",
		OwnerHelp:           "個人アカウントまたは Organization",
		OwnerPersonal:       "（個人アカウント）",
		VisibilityTitle:     "可視性",
		VisibilityPrivate:   "Private",
		VisibilityPublic:    "Public",
		VisibilityInternal:  "Internal",

		InitGroupTitle: "初期化",
		AddReadmeTitle: "README を追加する",
		GitignoreTitle: ".gitignore を追加する",
		GitignoreHelp:  ".gitignore テンプレートを選択",
		GitignoreNone:  "なし",
		LicenseTitle:   "ライセンスを選択",
		LicenseHelp:    "ライセンスはコードの利用条件を他者に示します",
		LicenseNone:    "なし",

		FeaturesGroupTitle: "機能",
		FeatIssues:         "Issues",
		FeatProjects:       "Projects",
		FeatWiki:           "Wikis",
		FeatDiscussions:    "Discussions",

		RulesetsGroupTitle:  "ルールセット",
		RulesetsHelp:        "作成後に適用する gh-rulekit の保存済みルールセット",
		RulesetsLabel:       "ルールセット",
		StepApplyingRuleset: "ルールセット %s を適用中",
		RulesetApplied:      "ルールセット %s を適用しました",
		RulekitNotInstalled: "gh-rulekit が未インストールのため、ルールセット %d 件をスキップします",

		PRGroupTitle:          "プルリクエスト",
		PRAllowMergeCommit:    "マージコミットを許可",
		PRAllowSquashMerge:    "スカッシュマージを許可",
		PRAllowRebaseMerge:    "リベースマージを許可",
		PRAllowAutoMerge:      "自動マージを許可",
		PRDeleteBranchOnMerge: "マージ後にヘッドブランチを自動削除",
		PRAllowUpdateBranch:   "PR ブランチの更新を常に提案",
		PRSquashTitleTitle:    "スカッシュコミットの既定タイトル",
		PRSquashMessageTitle:  "スカッシュコミットの既定メッセージ",
		PRMergeTitleTitle:     "マージコミットの既定タイトル",
		PRMergeMessageTitle:   "マージコミットの既定メッセージ",
		PRCommitTitleNote:     "対応するマージ方式が有効なときのみ適用されます",

		ConfirmGroupTitle:   "確認",
		ConfirmCreate:       "このリポジトリを作成しますか？",
		SummaryHeader:       "以下のリポジトリを作成します:",
		Cancelled:           "キャンセルしました。",
		ProfileLabel:        "プロファイル",
		SummaryFromTemplate: "テンプレート %s から",
		SummaryFromSource:   "ローカルソース %s から",

		StepCreatingRepo:   "リポジトリを作成中",
		StepRepoCreated:    "リポジトリを作成しました",
		StepApplyingConfig: "設定を適用中",
		StepConfigApplied:  "設定を適用しました",
		StepFailed:         "失敗",
		AllDone:            "完了",
		PartialFailure:     "リポジトリ %s は作成されましたが、一部の設定に失敗しました。再実行: gh new-repo apply %s",

		ProfileNotFound:      "プロファイル %q が設定に見つかりません",
		NoEditorEnv:          "設定を編集するには $EDITOR か $VISUAL を設定してください",
		ProfileSettingsTitle: "プロファイル設定",
		SaveProfileConfirm:   "このプロファイルを保存しますか？",
		ProfileCreated:       "プロファイル %q を作成しました",
		ProfileUpdated:       "プロファイル %q を更新しました",
		ProfileExists:        "プロファイル %q は既に存在します（`gh new-repo profile edit` で編集）",
		ProfileFromNotFound:  "コピー元プロファイル %q が見つかりません",
		DefaultProfileSet:    "既定プロファイルを %q に設定しました",
		NoProfilesHint:       "プロファイルがありません。`gh new-repo profile new <name>` で作成してください",

		LanguageSettingTitle: "言語",
		LanguageSettingHelp:  "gh-new-repo 自身のプロンプトとメッセージの言語",
		LangAuto:             "自動（LANG から判定）",
		LangEnglish:          "English",
		LangJapanese:         "日本語",
		LanguageSet:          "言語を %s に設定しました",
		InvalidLanguage:      "言語は auto / en / ja のいずれかを指定してください",
	}
)

// For は、指定された言語の文言セットを返します。
func For(l Lang) Strings {
	if l == JA {
		return ja
	}
	return en
}

// Detect は、明示指定と環境変数から表示言語を決定します。
// pref には、設定ファイルまたは --lang の値（"auto"、""、"en"、"ja"）を渡します。
func Detect(pref string) Lang {
	switch strings.ToLower(strings.TrimSpace(pref)) {
	case "ja", "japanese", "日本語":
		return JA
	case "en", "english":
		return EN
	}
	for _, key := range []string{"GH_NEW_REPO_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.ToLower(os.Getenv(key))
		if v == "" {
			continue
		}
		if strings.HasPrefix(v, "ja") {
			return JA
		}
		if strings.HasPrefix(v, "en") {
			return EN
		}
	}
	return EN
}
