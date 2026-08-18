.PHONY: build backend run clean test test-integration bench frontend frontend-dev all deps

# Pass VERSION=vX.Y.Z for official releases. Development builds intentionally
# retain the prerelease suffix so GitHub update checks can identify stable tags.
VERSION ?= v2.0.0-dev
GO ?= go
BUILD_LDFLAGS := -X zhiyuwaf/internal/dashboard.BuildVersion=$(VERSION)

backend:
	rm -rf internal/dashboard/dist
	mkdir -p internal/dashboard
	cp -R web/dist internal/dashboard/dist
	$(GO) build -ldflags "$(BUILD_LDFLAGS)" -o bin/zhiyu-waf ./cmd/zhiyu-waf

frontend:
	cd web && npm run build

frontend-dev:
	cd web && npm run dev

build: frontend backend

all: build

run: build
	sudo ./bin/zhiyu-waf -config configs/zhiyu-waf.yaml

test:
	go test ./...

test-integration:
	go test -v -count=1 ./internal/dashboard/ -run TestIntegration

bench:
	go test -bench=. -benchmem -count=3 ./internal/engine/ ./internal/proxy/

clean:
	rm -rf bin/ data/ web/dist/ internal/dashboard/dist/

deps:
	go mod tidy
	cd web && npm install
