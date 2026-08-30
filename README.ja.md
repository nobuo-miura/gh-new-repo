# gh-new-repo

[English](README.md) | [日本語](README.ja.md)

GitHub リポジトリを作成するたびに、可視性やマージ方式などを設定し直す手間を減らす [`gh`](https://cli.github.com/) 拡張機能です。

よく使う設定を名前付きプロファイルとして保存し、対話フォームまたは1つのコマンドから再利用できます。リポジトリの作成は公式の `gh repo create` に委譲し、Projects、Discussions、Pull Request のマージ設定などは作成後に GitHub API で適用します。

## 主な機能

- 可視性、機能、Pull Request のマージ設定などを YAML プロファイルとして保存
- プロファイルの値を入力済みの状態で、リポジトリ作成フォームを表示
- リポジトリ名を指定した非対話実行と、実行前の確認に対応
- `--source`、`--template`、`--clone`、`--push` など、主要な `gh repo create` フラグをサポート
- 既存リポジトリにもプロファイルを適用可能
- UI は英語と日本語に対応
- オプションで [`gh-rulekit`](https://github.com/nobuo-miura/gh-rulekit) の保存済みルールセットを適用

## スクリーンショット

`gh new-repo <name>` は、作成前にサマリを表示して確認を求めます。

![確認サマリ](images/ja-create.jpeg)

名前なし（または `--interactive`）で実行すると、すべての設定をフォームで確認できます。

![対話フォーム](images/ja-set.jpeg)

<details><summary>英語 UI</summary>

![英語版の確認サマリ](images/en-create.jpeg)

![英語版の対話フォーム](images/en-set.jpeg)

</details>

## 必要要件

- [`gh`](https://cli.github.com/) がインストール済みで、`gh auth login` などによる認証が完了していること
- ルールセット連携を使用する場合のみ、[`gh-rulekit`](https://github.com/nobuo-miura/gh-rulekit) がインストール済みであること

追加のトークンや API キーは必要ありません。

## インストール

```sh
gh extension install nobuo-miura/gh-new-repo
```

このコマンドは、GitHub Releases で公開された最新のビルド済みバイナリをインストールします。最初のリリースが公開されるまでは、後述の[開発版のインストール](#開発)を使用してください。

## クイックスタート

最初にプロファイルを作成します。組み込みプロファイルはないため、初回のみ対話フォームで設定してください。最初に作成したプロファイルは自動的に既定になります。

```sh
gh new-repo profile new personal
```

以降は、リポジトリ名を指定するだけで既定プロファイルを再利用できます。作成前に設定のサマリと確認プロンプトが表示されます。

```sh
gh new-repo my-repo
```

すべての項目をフォームで確認・変更したい場合は、リポジトリ名を省略します。

```sh
gh new-repo
```

## 使い方

### リポジトリを作成する

```sh
# 既定プロファイルで作成
gh new-repo my-repo

# プロファイルを指定
gh new-repo my-repo --profile oss

# Organization 配下に作成
gh new-repo my-org/my-repo --profile oss

# フラグでプロファイルの初期値を上書き
gh new-repo my-repo --public -d "説明" -g Go -l mit --clone

# ローカルリポジトリをソースとして作成し、コミットを push
gh new-repo my-repo --source=. --push

# 名前を指定したままフォームを開く
gh new-repo my-repo --interactive

# 確認を省略
gh new-repo my-repo --yes

# 実行内容だけを表示
gh new-repo my-repo --dry-run
```

| 実行方法 | 動作 |
|---|---|
| `gh new-repo` | 必要に応じてプロファイルを選択し、作成フォームを表示 |
| `gh new-repo <name>` | プロファイルを適用し、サマリと確認プロンプトを表示 |
| `gh new-repo <name> --yes` | 確認プロンプトを省略して作成 |
| `gh new-repo <name> --interactive` | リポジトリ名を入力済みの状態でフォームを表示 |

### `gh repo create` 互換フラグ

次のフラグをサポートしています。

`--public` `--private` `--internal` `-d/--description` `-g/--gitignore` `-l/--license` `--add-readme` `--disable-issues` `--disable-wiki` `--homepage` `-t/--team` `-c/--clone` `-s/--source` `--push` `-r/--remote` `-p/--template` `--include-all-branches`

- 明示したフラグはプロファイルの値より優先されます。例えば、`--description ""` でプロファイルの説明を空にできます。
- `--public`、`--private`、`--internal` は同時に指定できません。
- `--source` または `--template` を指定した場合、プロファイルの README、`.gitignore`、ライセンス設定は渡しません。
- `-p` は `--template` の短縮形です。プロファイルの指定には `--profile` を使用します。
- `--dry-run` は `gh repo create` の引数、作成後の API リクエストボディ、ルールセット適用コマンドを表示します。GitHub への変更は行いません。

### プロファイルを管理する

```sh
gh new-repo profile new dev                 # 新しいプロファイルを作成
gh new-repo profile new dev --from oss      # 既存プロファイルを初期値として使用
gh new-repo profile new dev --set-default   # 作成と同時に既定へ設定
gh new-repo profile default oss             # 既定プロファイルを変更
gh new-repo profile list                    # プロファイル一覧を表示
gh new-repo profile show oss                # YAML として表示
gh new-repo profile edit oss                # フォームで編集
gh new-repo profile edit                    # 既定プロファイルを編集
gh new-repo profile edit --file             # 設定ファイルをエディターで開く
gh new-repo profile path                    # 設定ファイルのパスを表示
```

`profile list` では、既定プロファイルに `*` が表示されます。`profile edit --file` は `GH_NEW_REPO_EDITOR`、`VISUAL`、`EDITOR` の順に使用するエディターを決定します。

### 既存リポジトリに適用する

```sh
gh new-repo apply owner/repo --profile personal
```

`apply` はリポジトリを作成せず、プロファイルで明示された機能、Pull Request 設定、ルールセットだけを既存リポジトリへ適用します。README、`.gitignore`、ライセンスなどの初期化設定は対象外です。

### 表示言語を設定する

```sh
gh new-repo config language          # 自動 / English / 日本語から選択
gh new-repo config language ja       # auto | en | ja を直接指定
gh new-repo config show              # 現在の設定を表示
```

既定値の `auto` は `LANG` などの環境変数から言語を判定します。`--lang en` または `--lang ja` を指定すると、その実行だけ表示言語を変更できます。

## 設定ファイル

設定は `$XDG_CONFIG_HOME/gh-new-repo/config.yml` に保存されます。`XDG_CONFIG_HOME` が未設定の場合は、OS 標準の設定ディレクトリを使用します。ファイルは、プロファイルの作成や言語設定など、最初の変更時に作成されます。

```yaml
language: auto
default_profile: personal
profiles:
  personal:
    description: ""
    visibility: private
    features:
      issues: true
      projects: false
      wiki: false
      discussions: false
    pull_requests:
      allow_merge_commit: false
      allow_squash_merge: true
      allow_rebase_merge: false
      allow_auto_merge: true
      delete_branch_on_merge: true
      allow_update_branch: true
      squash_merge_commit_title: PR_TITLE
      squash_merge_commit_message: PR_BODY
    init:
      add_readme: false
      gitignore_template: ""
      license_template: ""
    rulesets:
      - protect-default
```

- `features` と `pull_requests` の真偽値を省略すると、その設定には触れません。既存リポジトリへ `apply` するときに、現在の値を維持したい項目を除外できます。
- 対話フォームで保存すると、フォームに表示された真偽値はすべて明示的に記録されます。
- `init` は新規リポジトリの作成時だけ使用します。

## gh-rulekit 連携

`gh-rulekit` をインストールすると、保存済みルールセットをプロファイルに関連付け、リポジトリの作成後または `apply` の実行時に適用できます。

```sh
gh extension install nobuo-miura/gh-rulekit
gh rulekit export owner/some-repo --name main --as protect-default
```

`gh new-repo`、`profile new`、`profile edit` のフォームにルールセットのチェックリストが表示されます。`gh-rulekit` を利用できない場合はチェックリストを表示せず、プロファイルの `rulesets` をスキップします。

## 対応している設定

| グループ | 設定 | 適用方法 |
|---|---|---|
| General | 説明、可視性 | `gh repo create` |
| Init | README、`.gitignore`、ライセンス | `gh repo create` |
| Features | Issues、Wikis | `gh repo create` と GitHub API |
| Features | Projects、Discussions | GitHub API |
| Pull Requests | マージ方式、コミット文言、auto-merge、マージ後のブランチ削除、ブランチ更新提案 | GitHub API |
| Rulesets | ブランチ、タグのルールセット | `gh-rulekit` |

### 現在対応していない設定

- Actions の権限（`default_workflow_permissions` など）
- セキュリティ機能
- プロファイルによるコラボレーターとチームの管理（作成時の `--team` 指定には対応）
- トピック
- Issue を作成できるユーザーの範囲

## 開発

```sh
make check
```

`make check` はビルド、テスト、`gofmt`、`go vet`、`golangci-lint` を実行します。

ローカルの開発版を `gh` 拡張機能として登録する場合は、次のコマンドを使用します。

```sh
git clone https://github.com/nobuo-miura/gh-new-repo.git
cd gh-new-repo
make install
```

`make install` はバイナリをビルドしてから、このリポジトリをシンボリックリンク形式でインストールします。コードを変更した後は `make build` で再ビルドしてください。

## ライセンス

[MIT License](LICENSE)
