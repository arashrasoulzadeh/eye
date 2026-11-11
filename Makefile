.PHONY: server admin test admin-build mutation-test
server:
	cd server && go run ./cmd/server --env=../.env

admin:
	cd admin && npm run dev -- --webpack

test:
	go test ./tests/...
	cd admin && npm run build -- --webpack

admin-build:
	cd admin && npm run build -- --webpack

mutation-test:
	@command -v go-mutesting >/dev/null 2>&1 || { echo "go-mutesting is required; install with: go install github.com/avito-tech/go-mutesting/cmd/go-mutesting@bb372ba"; exit 1; }
	go-mutesting --match='ValidateForProduction' ./internal/config
