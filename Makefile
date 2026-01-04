GO = go

.PHONY: clean default dev lint

default: dev

clean:
	@$(GO) clean
	@rm -rf bin

dev:
	@env $(shell grep -v ^# .env | xargs) \
	$(GO) run cmd/reader/*

lint:
	@$(GO) fmt $(shell $(GO) list ./... | grep -v /vendor/)
