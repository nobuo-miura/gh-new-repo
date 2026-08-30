// gh-new-repo は、保存した設定プロファイルを使って GitHub リポジトリを作成する
// gh 拡張機能です。
package main

import (
	"fmt"
	"os"

	"github.com/nobuo-miura/gh-new-repo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if msg := err.Error(); msg != "" {
			fmt.Fprintln(os.Stderr, "gh-new-repo:", msg)
		}
		os.Exit(1)
	}
}
