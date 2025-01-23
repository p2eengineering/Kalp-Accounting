
.PHONY: help test cover

help:
	@echo "Usage:"
	@echo "  make <target>"
	@echo "Targets:"
	@echo "  test          Runs all the tests in the sub folders."
	@echo "  cover         Opens the html which shows the coverage report"

test:
	rm -f coverage.out
	go test -coverprofile=coverage.out ./...

cover:
	go tool cover -html=coverage.out