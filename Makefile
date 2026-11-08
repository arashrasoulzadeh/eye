.PHONY: server admin test
server:
	cd server && go run ./cmd/server --env=.env

admin:
	cd admin && npm run dev

test:
	go test ./server/test/...
	cd admin && npm run build -- --webpack
