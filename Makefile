.DEFAULT_GOAL := help

APP =

.PHONY: help
help: ## Display this help screen
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_\/-]+:.*##/ {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build-test
test: ## Builds and tests using maelstrom
	GOOS=linux GOARCH=amd64 go build -o bin/$(APP) ./cmd/$(APP)
	docker run --rm \
		-v '$(shell pwd):/app' \
		maelstrom \
		maelstrom test -w $(APP) --bin ./bin/$(APP) $(ARGS)
