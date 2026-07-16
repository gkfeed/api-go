.PHONY: check dev format lint merge-to-master test update

update:
	git fetch && git pull
	docker compose stop && docker compose rm -f
	docker compose build
	docker compose up -d

test:
	uv run pytest

check:
	uv run ruff format --check .
	uv run ruff check .
	uv run pytest

lint:
	uv run ruff check .

format:
	uv run ruff check --fix .
	uv run ruff format .

dev:
	uv run uvicorn gkfeed.api:app --reload --host 0.0.0.0 --port 8086

merge-to-master:
	git checkout master
	git merge dev
	git push origin master
	git checkout dev
