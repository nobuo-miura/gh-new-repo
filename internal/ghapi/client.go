// Package ghapi は、gh-new-repo で使用する GitHub REST API クライアントを提供します。
package ghapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Client は、認証済みの GitHub REST API クライアントです。
type Client struct {
	rest *api.RESTClient
}

// New は、gh の認証情報を使用して Client を作成します。
func New() (*Client, error) {
	rest, err := api.DefaultRESTClient()
	if err != nil {
		return nil, err
	}
	return &Client{rest: rest}, nil
}

// Login は、現在認証されているユーザーのログイン名を返します。
func (c *Client) Login() (string, error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := c.rest.Get("user", &user); err != nil {
		return "", err
	}
	return user.Login, nil
}

// Orgs は、現在のユーザーが所属する Organization のログイン名を返します。
func (c *Client) Orgs() ([]string, error) {
	var orgs []struct {
		Login string `json:"login"`
	}
	if err := c.rest.Get("user/orgs?per_page=100", &orgs); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(orgs))
	for _, o := range orgs {
		names = append(names, o.Login)
	}
	sort.Strings(names)
	return names, nil
}

// GitignoreTemplates は、利用可能な .gitignore テンプレート名を返します。
func (c *Client) GitignoreTemplates() ([]string, error) {
	var names []string
	if err := c.rest.Get("gitignore/templates", &names); err != nil {
		return nil, err
	}
	return names, nil
}

// License は、選択可能なライセンスを表します。
type License struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// Licenses は、選択可能なライセンスの一覧を返します。
func (c *Client) Licenses() ([]License, error) {
	var licenses []License
	if err := c.rest.Get("licenses", &licenses); err != nil {
		return nil, err
	}
	return licenses, nil
}

// Repo は、リポジトリ API のレスポンスで使用する項目を表します。
type Repo struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	HTMLURL       string `json:"html_url"`
	DefaultBranch string `json:"default_branch"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// CreateRepo はリポジトリを作成します。org が空の場合は個人アカウント配下に作成します。
func (c *Client) CreateRepo(org string, body map[string]any) (*Repo, error) {
	path := "user/repos"
	if org != "" {
		path = fmt.Sprintf("orgs/%s/repos", org)
	}
	var repo Repo
	if err := c.post(path, body, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// UpdateRepo は、PATCH /repos/{owner}/{repo} でリポジトリ設定を更新します。
func (c *Client) UpdateRepo(owner, name string, body map[string]any) error {
	return c.patch(fmt.Sprintf("repos/%s/%s", owner, name), body, nil)
}

// GetRepo は、既存リポジトリの情報を取得します。
func (c *Client) GetRepo(owner, name string) (*Repo, error) {
	var repo Repo
	if err := c.rest.Get(fmt.Sprintf("repos/%s/%s", owner, name), &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

func (c *Client) post(path string, body map[string]any, out any) error {
	return c.do(http.MethodPost, path, body, out)
}

func (c *Client) patch(path string, body map[string]any, out any) error {
	return c.do(http.MethodPatch, path, body, out)
}

func (c *Client) do(method, path string, body map[string]any, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := c.rest.Request(method, path, bytes.NewReader(buf))
	if err != nil {
		return apiError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, _ := io.ReadAll(resp.Body)
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return err
		}
	}
	return nil
}

// apiError は、go-gh の HTTPError を利用者向けのエラーメッセージへ変換します。
func apiError(err error) error {
	var httpErr *api.HTTPError
	if !errors.As(err, &httpErr) {
		return err
	}
	msg := httpErr.Message
	if msg == "" {
		msg = http.StatusText(httpErr.StatusCode)
	}
	parts := make([]string, 0, len(httpErr.Errors))
	for _, e := range httpErr.Errors {
		switch {
		case e.Message != "":
			parts = append(parts, e.Message)
		case e.Field != "":
			parts = append(parts, fmt.Sprintf("%s: %s", e.Field, e.Code))
		}
	}
	if len(parts) > 0 {
		return fmt.Errorf("%s (%s)", msg, strings.Join(parts, "; "))
	}
	return errors.New(msg)
}
