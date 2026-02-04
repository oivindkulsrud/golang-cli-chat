.PHONY: help run install

BINARY_NAME=chat-cli

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-10s %s\n", $$1, $$2}'

run: ## Build and run the chat application
	go build -o $(BINARY_NAME) .
	bash -c "source .env && ./$(BINARY_NAME)"

install: ## Install Go dependencies
	go mod download
	go mod tidy

fmt: ## Format the code
	go fmt
