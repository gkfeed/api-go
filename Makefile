update:
	git fetch && git pull
	docker compose stop && docker compose rm -f
	docker compose build
	docker compose up -d

test:
	cd app && go test ./...

lint:
	cd app && $(shell go env GOPATH)/bin/golangci-lint run ./...

format:
	cd app && go fmt ./...

install-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

dev:
	cd app && go run ./cmd/api/main.go

merge-to-master:
	git checkout master
	git merge dev
	git push origin master
	git checkout dev