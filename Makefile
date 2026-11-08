.PHONY: server admin test
server:
	cd server && go run ./cmd/server --env=.env

admin:
	cd admin && npm run build -- --webpack

test:
	go test ./tests/...
	cd admin && npm run build -- --webpack
