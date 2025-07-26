run:
	go run ./cmd/trac2gitlab

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
