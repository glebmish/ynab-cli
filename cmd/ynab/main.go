package main

import (
	"errors"
	"os"

	"github.com/glebmish/ynab-cli/internal/api"
	"github.com/glebmish/ynab-cli/internal/cliexit"
	"github.com/glebmish/ynab-cli/internal/cmd"
)

// version is set at build time via -ldflags "-X main.version=...".
// GoReleaser injects the git tag; a plain `go build` leaves it as "dev".
var version = "dev"

// main does not print the error — cobra already does (SilenceUsage:true,
// SilenceErrors:false). Printing here on top would double every error.
func main() {
	cmd.SetVersion(version)
	if err := cmd.Execute(); err != nil {
		os.Exit(exitCode(err))
	}
}

func exitCode(err error) int {
	var authErr *cliexit.AuthError
	if errors.As(err, &authErr) {
		return 2
	}
	var valErr *cliexit.ValidationError
	if errors.As(err, &valErr) {
		return 3
	}
	var discErr *cliexit.DiscoveryError
	if errors.As(err, &discErr) {
		return 4
	}
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		if apiErr.IsAuth() {
			return 2
		}
		return 1
	}
	return 1
}
