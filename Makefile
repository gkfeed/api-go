.PHONY: check dev format install-lint lint merge-to-master test update vet

APP_DIR := app

update:
	git fetch && git pull
	docker compose stop && docker compose rm -f
	docker compose build
	docker compose up -d

test:
	cd $(APP_DIR) && go test ./...

vet:
	cd $(APP_DIR) && go vet ./...

check: test vet

lint:
	cd $(APP_DIR) && golangci-lint run ./...

format:
	cd $(APP_DIR) && go fmt ./...

install-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

dev:
	cd $(APP_DIR) && go run ./cmd/api

merge-to-master:
	git checkout master
	git merge dev
	git push origin master
	git checkout dev
