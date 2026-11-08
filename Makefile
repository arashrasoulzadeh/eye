.PHONY: server admin test
server:
	cd server && go run ./cmd/server --env=.env

admin:
	cd admin && npm run dev

test:
	go test ./tests/...
	cd admin && npm run build -- --webpack

admin-build:
	cd admin && npm run build -- --webpack
