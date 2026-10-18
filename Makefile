GOVULNCHECK_VERSION ?= v1.9.0
BIN := bin/agentguard

.PHONY: all fmt fmt-check vet lint vuln test build clean

all: fmt-check vet lint test build

fmt:
	gofmt -w .

fmt-check:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

lint:
	golangci-lint run

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

test:
	go test -race ./...

build:
	go build -trimpath -o $(BIN) ./cmd/agentguard

clean:
	rm -rf bin
