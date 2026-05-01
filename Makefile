.PHONY: install run generate clean

generate:
	go generate ./internal/server/...

run: generate
	go run . --ps 192.168.1.100

install: generate
	go install -ldflags="-s -w" -trimpath .

clean:
	rm -rf internal/server/web
