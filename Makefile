.PHONY: build

GOBIN = ./build/bin
GO ?= latest

build:
	go build -o build/node-payout-manager
	cp config.json build/config.json
	@echo "Done building."
