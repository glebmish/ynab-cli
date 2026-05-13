package cmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/glebmish/ynab-cli/internal/cliexit"
	"github.com/spf13/cobra"
)

//go:embed ynab-api.json
var specData []byte

func discoveryErr(err error) error {
	if err == nil {
		return nil
	}
	return &cliexit.DiscoveryError{Err: err}
}

type operationEntry struct {
	CommandName string
	OperationID string
	Method      string
	Path        string
	PathItem    map[string]interface{}
}

// operationIDToCommand maps YNAB operationId → CLI command name.
var operationIDToCommand = map[string]string{
	"getUser":                       "user.get",
	"getPlans":                      "plans.list",
	"getPlanById":                   "plans.get",
	"getPlanSettingsById":           "plans.get-settings",
	"getAccounts":                   "accounts.list",
	"createAccount":                 "accounts.create",
	"getAccountById":                "accounts.get",
	"getCategories":                 "categories.list",
	"createCategory":                "categories.create",
	"getCategoryById":               "categories.get",
	"updateCategory":                "categories.update",
	"getMonthCategoryById":          "categories.get-month",
	"updateMonthCategory":           "categories.update-month",
	"createCategoryGroup":           "categories.create-group",
	"updateCategoryGroup":           "categories.update-group",
	"getPayees":                     "payees.list",
	"createPayee":                   "payees.create",
	"getPayeeById":                  "payees.get",
	"updatePayee":                   "payees.update",
	"getPayeeLocations":             "payee-locations.list",
	"getPayeeLocationById":          "payee-locations.get",
	"getPayeeLocationsByPayee":      "payee-locations.list-by-payee",
	"getPlanMonths":                 "months.list",
	"getPlanMonth":                  "months.get",
	"getMoneyMovements":             "money-movements.list",
	"getMoneyMovementsByMonth":      "money-movements.list-by-month",
	"getMoneyMovementGroups":        "money-movements.list-groups",
	"getMoneyMovementGroupsByMonth": "money-movements.list-groups-by-month",
	"getTransactions":               "transactions.list",
	"createTransaction":             "transactions.create",
	"updateTransactions":            "transactions.update-bulk",
	"importTransactions":            "transactions.import",
	"getTransactionById":            "transactions.get",
	"updateTransaction":             "transactions.update",
	"deleteTransaction":             "transactions.delete",
	"getTransactionsByAccount":      "transactions.list-by-account",
	"getTransactionsByCategory":     "transactions.list-by-category",
	"getTransactionsByPayee":        "transactions.list-by-payee",
	"getTransactionsByMonth":        "transactions.list-by-month",
	"getScheduledTransactions":      "scheduled-transactions.list",
	"createScheduledTransaction":    "scheduled-transactions.create",
	"getScheduledTransactionById":   "scheduled-transactions.get",
	"updateScheduledTransaction":    "scheduled-transactions.update",
	"deleteScheduledTransaction":    "scheduled-transactions.delete",
}

var schemaCmd = &cobra.Command{
	Use:   "schema [operation|type]",
	Short: "Inspect API operation schemas or type definitions",
	Long: `Inspect the YNAB API schema.

  ynab schema transactions.list     # Show operation schema
  ynab schema TransactionDetail     # Show type definition
  ynab schema --list                # List all operations`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		listFlag, _ := cmd.Flags().GetBool("list")
		resolveRefs, _ := cmd.Flags().GetBool("resolve-refs")

		spec, err := parseSpec()
		if err != nil {
			return err
		}

		if listFlag {
			return listOperations(spec)
		}

		if len(args) == 0 {
			return discoveryErr(fmt.Errorf("provide an operation path (e.g., transactions.list) or type name (e.g., TransactionDetail)\n  Use --list to see all operations"))
		}

		path := args[0]
		if !strings.Contains(path, ".") {
			return showType(spec, path, resolveRefs)
		}
		return showOperation(spec, path, resolveRefs)
	},
}

func init() {
	schemaCmd.Flags().Bool("list", false, "List all available operations")
	schemaCmd.Flags().Bool("resolve-refs", false, "Inline $ref references in output")
	rootCmd.AddCommand(schemaCmd)
}

func parseSpec() (map[string]interface{}, error) {
	var spec map[string]interface{}
	if err := json.Unmarshal(specData, &spec); err != nil {
		return nil, discoveryErr(fmt.Errorf("parsing embedded OpenAPI spec: %w", err))
	}
	return spec, nil
}

func parseSchemaPath(path string) (string, string, error) {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) != 2 {
		return "", "", discoveryErr(fmt.Errorf("schema path must be 'resource.action' (e.g., transactions.list), got %q", path))
	}
	return parts[0], parts[1], nil
}

func buildOperationIndex(spec map[string]interface{}) map[string]operationEntry {
	index := make(map[string]operationEntry)
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		return index
	}
	for apiPath, methods := range paths {
		methodMap, ok := methods.(map[string]interface{})
		if !ok {
			continue
		}
		for method, opRaw := range methodMap {
			switch method {
			case "get", "post", "put", "patch", "delete":
			default:
				continue
			}
			op, ok := opRaw.(map[string]interface{})
			if !ok {
				continue
			}
			operationID, _ := op["operationId"].(string)
			cmdName, ok := operationIDToCommand[operationID]
			if !ok {
				continue
			}
			index[cmdName] = operationEntry{
				CommandName: cmdName,
				OperationID: operationID,
				Method:      strings.ToUpper(method),
				Path:        apiPath,
				PathItem:    op,
			}
		}
	}
	return index
}

func buildSchemaOutput(spec map[string]interface{}, entry operationEntry) map[string]interface{} {
	output := map[string]interface{}{
		"command":     entry.CommandName,
		"operationId": entry.OperationID,
		"method":      entry.Method,
		"path":        entry.Path,
	}
	if desc, ok := entry.PathItem["description"].(string); ok {
		output["description"] = desc
	}
	if summary, ok := entry.PathItem["summary"].(string); ok {
		output["summary"] = summary
	}
	if params, ok := entry.PathItem["parameters"].([]interface{}); ok {
		paramList := []map[string]interface{}{}
		for _, p := range params {
			param, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			pe := map[string]interface{}{
				"name": param["name"], "in": param["in"], "required": param["required"],
			}
			if schema, ok := param["schema"].(map[string]interface{}); ok {
				pe["type"] = schema["type"]
				if f, ok := schema["format"]; ok {
					pe["format"] = f
				}
			}
			if desc, ok := param["description"].(string); ok {
				pe["description"] = desc
			}
			paramList = append(paramList, pe)
		}
		output["parameters"] = paramList
	}
	if reqBody, ok := entry.PathItem["requestBody"].(map[string]interface{}); ok {
		if content, ok := reqBody["content"].(map[string]interface{}); ok {
			for ct, schemaRaw := range content {
				schema, ok := schemaRaw.(map[string]interface{})
				if !ok {
					continue
				}
				bodyInfo := map[string]interface{}{"contentType": ct}
				if s, ok := schema["schema"].(map[string]interface{}); ok {
					bodyInfo["schema"] = s
				}
				output["requestBody"] = bodyInfo
				break
			}
		}
	}
	if responses, ok := entry.PathItem["responses"].(map[string]interface{}); ok {
		respInfo := map[string]interface{}{}
		for code, respRaw := range responses {
			resp, ok := respRaw.(map[string]interface{})
			if !ok {
				continue
			}
			e := map[string]interface{}{}
			if desc, ok := resp["description"].(string); ok {
				e["description"] = desc
			}
			if content, ok := resp["content"].(map[string]interface{}); ok {
				for ct, schemaRaw := range content {
					s, ok := schemaRaw.(map[string]interface{})
					if !ok {
						continue
					}
					e["contentType"] = ct
					if schema, ok := s["schema"].(map[string]interface{}); ok {
						e["schema"] = schema
					}
					break
				}
			}
			respInfo[code] = e
		}
		output["responses"] = respInfo
	}
	return output
}

func showOperation(spec map[string]interface{}, path string, resolveRefs bool) error {
	index := buildOperationIndex(spec)
	entry, ok := index[path]
	if !ok {
		var suggestions []string
		resource, _, _ := parseSchemaPath(path)
		for name := range index {
			if strings.HasPrefix(name, resource+".") {
				suggestions = append(suggestions, name)
			}
		}
		sort.Strings(suggestions)
		msg := fmt.Sprintf("operation %q not found", path)
		if len(suggestions) > 0 {
			msg += fmt.Sprintf("\n  Available %s operations: %s", resource, strings.Join(suggestions, ", "))
		}
		return discoveryErr(fmt.Errorf("%s", msg))
	}
	output := buildSchemaOutput(spec, entry)
	if resolveRefs {
		components, _ := spec["components"].(map[string]interface{})
		schemas, _ := components["schemas"].(map[string]interface{})
		if schemas != nil {
			resolveAllRefs(output, schemas, map[string]bool{})
		}
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func showType(spec map[string]interface{}, name string, resolveRefs bool) error {
	components, ok := spec["components"].(map[string]interface{})
	if !ok {
		return discoveryErr(fmt.Errorf("no components in spec"))
	}
	schemas, ok := components["schemas"].(map[string]interface{})
	if !ok {
		return discoveryErr(fmt.Errorf("no schemas in spec"))
	}
	schema, ok := schemas[name]
	if !ok {
		var available []string
		for k := range schemas {
			available = append(available, k)
		}
		sort.Strings(available)
		return discoveryErr(fmt.Errorf("type %q not found\n  Available types: %s", name, strings.Join(available, ", ")))
	}
	output, ok := schema.(map[string]interface{})
	if !ok {
		return discoveryErr(fmt.Errorf("invalid schema for type %q", name))
	}
	if resolveRefs {
		resolveAllRefs(output, schemas, map[string]bool{name: true})
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func listOperations(spec map[string]interface{}) error {
	index := buildOperationIndex(spec)
	var names []string
	for name := range index {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		entry := index[name]
		fmt.Printf("%-45s  %s %s\n", name, entry.Method, entry.Path)
	}
	return nil
}

func resolveAllRefs(v interface{}, schemas map[string]interface{}, seen map[string]bool) {
	switch val := v.(type) {
	case map[string]interface{}:
		if ref, ok := val["$ref"].(string); ok {
			refName := strings.TrimPrefix(ref, "#/components/schemas/")
			if !seen[refName] {
				if schema, ok := schemas[refName].(map[string]interface{}); ok {
					seen[refName] = true
					for k, sv := range schema {
						if k != "$ref" {
							val[k] = deepCopy(sv)
						}
					}
					resolveAllRefs(val, schemas, seen)
					delete(seen, refName)
				}
			}
		}
		for _, v := range val {
			resolveAllRefs(v, schemas, seen)
		}
	case []interface{}:
		for _, item := range val {
			resolveAllRefs(item, schemas, seen)
		}
	}
}

func deepCopy(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		cp := make(map[string]interface{}, len(val))
		for k, v := range val {
			cp[k] = deepCopy(v)
		}
		return cp
	case []interface{}:
		cp := make([]interface{}, len(val))
		for i, v := range val {
			cp[i] = deepCopy(v)
		}
		return cp
	default:
		return v
	}
}
