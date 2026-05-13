# CLAUDE.md

## Project

Go CLI (`ynab`) for the YNAB (You Need A Budget) API. 44 operations across 10 resource groups.

## Build & Test

```bash
go build -o ynab .
go test ./...
go vet ./...
```

**Always install after building:** `go install . && cp ~/go/bin/ynab-cli ~/go/bin/ynab`

`go install .` names the binary after the module dir (`ynab-cli`), so copy to `ynab` to match the CLI name. Do this every time you make changes so the `ynab` binary on PATH stays current.

## Architecture

- `main.go` — entry point, calls `cmd.Execute()`
- `internal/cmd/` — Cobra commands. `root.go` wires config/client. `helpers.go` defines shared helpers (`doGet`, `doMutate`, `doDelete`). Each resource group is one file (`user.go`, `plans.go`, `accounts.go`, `categories.go`, `payees.go`, `payee_locations.go`, `months.go`, `money_movements.go`, `transactions.go`, `scheduled_transactions.go`).
- `internal/config/` — YAML config loading, env/flag override, validation (`YNAB_ACCESS_TOKEN`, `YNAB_PLAN_ID`, `YNAB_BASE_URL`, `YNAB_CONFIG`)
- `internal/api/` — HTTP client with Bearer auth, path substitution, error handling, envelope unwrap (`{"data": ...}`)
- `internal/validate/` — Input sanitization (path params, dates, JSON bodies)
- `internal/format/` — Output formatting (JSON, NDJSON, field filtering)
- `internal/cmd/ynab-api.json` — Embedded OpenAPI spec for the `schema` command
- `internal/cmd/skills/` — Embedded SKILL.md files; surface via `ynab skills list` / `ynab skills get <name>` (runtime) or `ynab skills install` (on disk)

## Adding a New API Operation

1. Identify the resource group file in `internal/cmd/`
2. Add a `new<Resource><Action>Cmd()` function following the existing pattern
3. Register it in the `init()` function's `AddCommand` call
4. Update the command name mapping in `schema.go` if needed
5. `go build` to verify

## Conventions

- All commands use shared helpers: `doGet`, `doMutate`, `doDelete`
- Required IDs validated with `validate.PathParam`, dates with `validate.DateParam`, months with `validateMonth` (accepts `YYYY-MM-DD` or the literal `current`)
- JSON bodies validated by the HTTP client before send
- API client retrieved from context: `api.FromContext(cmd.Context())`
- Path template variables: `{plan_id}` auto-substituted from config/flag. Other IDs are passed in the params map by the caller.
- All amounts in YNAB are **milliunits** (1/1000 of currency unit). Example: `$10.00 = 10000`.
- Responses are wrapped in `{"data": ...}`; the API client auto-unwraps before formatting.
