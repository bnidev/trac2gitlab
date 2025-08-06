run:
	go run ./cmd/trac2gitlab

init:
	go run ./cmd/trac2gitlab init

export:
	go run ./cmd/trac2gitlab export

migrate:
	go run ./cmd/trac2gitlab migrate

build:
	go build -o ./build/trac2gitlab ./cmd/trac2gitlab

test:
	go test -v -p=4 $(shell find . -type f -name '*_test.go' -exec dirname {} \; | sort -u)

clean:
	rm -rf build

format:
	goimports -l -w .

format-check:
	@goimports -l . | tee /dev/stderr

lint:
	golangci-lint run ./...

install:
	git config --local core.hooksPath .githooks || echo "Not in a Git repo?"
	go mod tidy
	go install golang.org/x/tools/cmd/goimports@v0.35.0
	command -v golangci-lint >/dev/null 2>&1 || \
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(HOME)/.local/bin v2.3.1
