// Package config は、gh-new-repo の設定ファイルとプロファイルを管理します。
//
// 設定ファイルは $XDG_CONFIG_HOME/gh-new-repo/config.yml に保存します。
// XDG_CONFIG_HOME が未設定の場合は、os.UserConfigDir が返すディレクトリを使用します。
// 真偽値を *bool で保持し、未指定の項目を変更しない状態を表現します。
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Features は、リポジトリ機能の設定を保持します。
type Features struct {
	Issues      *bool `yaml:"issues,omitempty"`
	Projects    *bool `yaml:"projects,omitempty"`
	Wiki        *bool `yaml:"wiki,omitempty"`
	Discussions *bool `yaml:"discussions,omitempty"`
}

// PullRequests は、Pull Request に関する設定を保持します。
// コミットタイトルとメッセージの既定値が空文字の場合、その項目は変更しません。
type PullRequests struct {
	AllowMergeCommit         *bool  `yaml:"allow_merge_commit,omitempty"`
	AllowSquashMerge         *bool  `yaml:"allow_squash_merge,omitempty"`
	AllowRebaseMerge         *bool  `yaml:"allow_rebase_merge,omitempty"`
	AllowAutoMerge           *bool  `yaml:"allow_auto_merge,omitempty"`
	DeleteBranchOnMerge      *bool  `yaml:"delete_branch_on_merge,omitempty"`
	AllowUpdateBranch        *bool  `yaml:"allow_update_branch,omitempty"`
	SquashMergeCommitTitle   string `yaml:"squash_merge_commit_title,omitempty"`
	SquashMergeCommitMessage string `yaml:"squash_merge_commit_message,omitempty"`
	MergeCommitTitle         string `yaml:"merge_commit_title,omitempty"`
	MergeCommitMessage       string `yaml:"merge_commit_message,omitempty"`
}

// Init は、リポジトリ作成時だけ使用する初期化オプションを保持します。
type Init struct {
	AddReadme         bool   `yaml:"add_readme,omitempty"`
	GitignoreTemplate string `yaml:"gitignore_template,omitempty"`
	LicenseTemplate   string `yaml:"license_template,omitempty"`
}

// Profile は、リポジトリ設定の名前付きプリセットです。
type Profile struct {
	Description  string       `yaml:"description,omitempty"`
	Visibility   string       `yaml:"visibility,omitempty"` // private | public | internal
	Features     Features     `yaml:"features"`
	PullRequests PullRequests `yaml:"pull_requests"`
	Init         Init         `yaml:"init"`
	// Rulesets は、gh-rulekit に保存されたルールセット名です。
	// リポジトリ作成後に `gh rulekit apply <name> --repo owner/repo` で適用し、
	// gh-rulekit を利用できない場合はスキップします。
	Rulesets []string `yaml:"rulesets,omitempty"`
}

// Config は、設定ファイル全体を表します。
type Config struct {
	Language       string             `yaml:"language,omitempty"` // auto | en | ja
	DefaultProfile string             `yaml:"default_profile,omitempty"`
	Profiles       map[string]Profile `yaml:"profiles,omitempty"`

	path string `yaml:"-"`
}

// ErrProfileNotFound は、指定されたプロファイルが存在しないことを示します。
var ErrProfileNotFound = errors.New("profile not found")

// Path は、設定ファイルの絶対パスを返します。
func Path() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "gh-new-repo", "config.yml"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gh-new-repo", "config.yml"), nil
}

// Load は設定ファイルを読み込みます。
// ファイルが存在しない場合は書き込みを行わず、空の設定を返します。
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{Language: "auto", Profiles: map[string]Profile{}, path: path}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	cfg.path = path
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return &cfg, nil
}

// Save は、現在の設定を設定ファイルへ書き込みます。
func (c *Config) Save() error {
	if c.path == "" {
		p, err := Path()
		if err != nil {
			return err
		}
		c.path = p
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}
	out, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, out, 0o644)
}

// Resolve は、名前を指定してプロファイルを取得します。
// name が空の場合は DefaultProfile を使用します。
func (c *Config) Resolve(name string) (string, Profile, error) {
	if name == "" {
		name = c.DefaultProfile
	}
	p, ok := c.Profiles[name]
	if !ok {
		return "", Profile{}, fmt.Errorf("%w: %s", ErrProfileNotFound, name)
	}
	return name, p, nil
}

// Names は、既定プロファイルを先頭にしたプロファイル名の一覧を返します。
// 既定プロファイル以外の順序は保証しません。
func (c *Config) Names() []string {
	names := make([]string, 0, len(c.Profiles))
	if _, ok := c.Profiles[c.DefaultProfile]; ok {
		names = append(names, c.DefaultProfile)
	}
	for name := range c.Profiles {
		if name != c.DefaultProfile {
			names = append(names, name)
		}
	}
	return names
}

// FilePath は、現在の設定ファイルのパスを返します。
func (c *Config) FilePath() string { return c.path }
