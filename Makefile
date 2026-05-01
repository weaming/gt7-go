.PHONY: install build run generate clean

generate:
	go generate ./internal/server/...

run: generate
	go run .

install: generate
	go install -ldflags="-s -w" -trimpath .

build: generate
	go build -o gt7-dashboard .

clean:
	rm -rf internal/server/web
