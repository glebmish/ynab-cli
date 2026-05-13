.PHONY: build install test

# YNAB's spec is checked in as ynab-api.json (already JSON), so there's no
# yaml→json `spec` target as in sibling CLIs. To refresh: replace ynab-api.json
# with the latest from https://api.ynab.com/papi/open_api_spec.yaml piped through
# yq -o=json '.' once if YNAB switches the canonical form to YAML.

test:
	go test ./...

build:
	go build -o ynab .

# `go install` names the binary after the module directory ("ynab-cli"), so we
# alias it to "ynab" to match the CLI's invocation name.
install: build
	go install .
	cp $$HOME/go/bin/ynab-cli $$HOME/go/bin/ynab
