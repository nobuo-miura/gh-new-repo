BINARY := gh-new-repo

.PHONY: build test lint check install

# ローカルの gh 拡張（symlink インストール）が参照するバイナリをビルドする
build:
	go build -o $(BINARY) .

test:
	go test ./...

lint:
	gofmt -l .
	go vet ./...
	golangci-lint run

check: build test lint

# このディレクトリを gh 拡張として登録し直す
install: build
	gh extension remove new-repo 2>/dev/null || true
	gh extension install .
