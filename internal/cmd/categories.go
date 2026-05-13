package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/glebmish/ynab-cli/internal/api"
	"github.com/glebmish/ynab-cli/internal/format"
	"github.com/glebmish/ynab-cli/internal/validate"
	"github.com/spf13/cobra"
)

var categoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "Categories and category groups",
}

func init() {
	categoriesCmd.AddCommand(
		newCategoriesListCmd(),
		newCategoriesCreateCmd(),
		newCategoriesGetCmd(),
		newCategoriesUpdateCmd(),
		newCategoriesGetMonthCmd(),
		newCategoriesUpdateMonthCmd(),
		newCategoriesCreateGroupCmd(),
		newCategoriesUpdateGroupCmd(),
	)
	rootCmd.AddCommand(categoriesCmd)
}

func newCategoriesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List categories (grouped)",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{}
			if v, _ := cmd.Flags().GetInt64("last-knowledge-of-server"); v > 0 {
				params["last_knowledge_of_server"] = fmt.Sprintf("%d", v)
			}
			return doGet(cmd, "/plans/{plan_id}/categories", params)
		},
	}
	cmd.Flags().Int64("last-knowledge-of-server", 0, "Server knowledge delta")
	return cmd
}

func newCategoriesCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a category",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonBody, err := requireJSON(cmd)
			if err != nil {
				return err
			}
			ifNotExists, _ := cmd.Flags().GetBool("if-not-exists")
			if ifNotExists {
				existing, err := findExistingCategory(cmd, jsonBody)
				if err != nil {
					return err
				}
				if existing != nil {
					wrapped, err := json.Marshal(map[string]interface{}{"category": existing})
					if err != nil {
						return err
					}
					return format.Write(os.Stdout, wrapped, fmtOpts(cmd))
				}
			}
			return doMutate(cmd, "POST", "/plans/{plan_id}/categories", nil, jsonBody)
		},
	}
	cmd.Flags().Bool("if-not-exists", false, "If a category with the same name in the target group already exists, return it instead of creating")
	return cmd
}

// findExistingCategory inspects the categories.create payload and returns
// the existing category with the same (name, category_group_id), if any.
// Returns (nil, nil) when no match.
func findExistingCategory(cmd *cobra.Command, jsonBody string) (map[string]interface{}, error) {
	var payload struct {
		Category struct {
			Name            string `json:"name"`
			CategoryGroupID string `json:"category_group_id"`
		} `json:"category"`
	}
	if err := json.Unmarshal([]byte(jsonBody), &payload); err != nil {
		return nil, fmt.Errorf("parsing --json for --if-not-exists: %w", err)
	}
	if payload.Category.Name == "" || payload.Category.CategoryGroupID == "" {
		return nil, nil
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		return nil, nil
	}

	c := api.FromContext(cmd.Context())
	resp, err := c.Do("GET", "/plans/{plan_id}/categories", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading categories list: %w", err)
	}

	unwrapped, err := format.UnwrapBody(body)
	if err != nil {
		return nil, fmt.Errorf("parsing categories list: %w", err)
	}
	groups, ok := unwrapped.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected categories list shape")
	}
	for _, gRaw := range groups {
		g, _ := gRaw.(map[string]interface{})
		if g == nil {
			continue
		}
		if gid, _ := g["id"].(string); gid != payload.Category.CategoryGroupID {
			continue
		}
		cats, _ := g["categories"].([]interface{})
		for _, c := range cats {
			cat, _ := c.(map[string]interface{})
			if cat == nil {
				continue
			}
			if name, _ := cat["name"].(string); name != payload.Category.Name {
				continue
			}
			if deleted, _ := cat["deleted"].(bool); deleted {
				continue
			}
			return cat, nil
		}
	}
	return nil, nil
}

func newCategoriesGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a category by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "category-id")
			if err != nil {
				return err
			}
			if err := validate.PathParam("category-id", id); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/categories/{category_id}", map[string]string{"category_id": id})
		},
	}
	cmd.Flags().String("category-id", "", "Category ID (required)")
	return cmd
}

func newCategoriesUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a category",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "category-id")
			if err != nil {
				return err
			}
			if err := validate.PathParam("category-id", id); err != nil {
				return err
			}
			jsonBody, err := requireJSON(cmd)
			if err != nil {
				return err
			}
			return doMutate(cmd, "PATCH", "/plans/{plan_id}/categories/{category_id}", map[string]string{"category_id": id}, jsonBody)
		},
	}
	cmd.Flags().String("category-id", "", "Category ID (required)")
	return cmd
}

func newCategoriesGetMonthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-month",
		Short: "Get a category for a specific month",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "category-id")
			if err != nil {
				return err
			}
			month, err := requireString(cmd, "month")
			if err != nil {
				return err
			}
			if err := validate.PathParam("category-id", id); err != nil {
				return err
			}
			if err := validateMonth(month); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/months/{month}/categories/{category_id}", map[string]string{"category_id": id, "month": month})
		},
	}
	cmd.Flags().String("category-id", "", "Category ID (required)")
	cmd.Flags().String("month", "", "Month (YYYY-MM-DD or 'current', required)")
	return cmd
}

func newCategoriesUpdateMonthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-month",
		Short: "Update a category for a specific month",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "category-id")
			if err != nil {
				return err
			}
			month, err := requireString(cmd, "month")
			if err != nil {
				return err
			}
			jsonBody, err := requireJSON(cmd)
			if err != nil {
				return err
			}
			if err := validate.PathParam("category-id", id); err != nil {
				return err
			}
			if err := validateMonth(month); err != nil {
				return err
			}
			return doMutate(cmd, "PATCH", "/plans/{plan_id}/months/{month}/categories/{category_id}",
				map[string]string{"category_id": id, "month": month}, jsonBody)
		},
	}
	cmd.Flags().String("category-id", "", "Category ID (required)")
	cmd.Flags().String("month", "", "Month (YYYY-MM-DD or 'current', required)")
	return cmd
}

func newCategoriesCreateGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-group",
		Short: "Create a category group",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonBody, err := requireJSON(cmd)
			if err != nil {
				return err
			}
			return doMutate(cmd, "POST", "/plans/{plan_id}/category_groups", nil, jsonBody)
		},
	}
	return cmd
}

func newCategoriesUpdateGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-group",
		Short: "Update a category group",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "category-group-id")
			if err != nil {
				return err
			}
			jsonBody, err := requireJSON(cmd)
			if err != nil {
				return err
			}
			if err := validate.PathParam("category-group-id", id); err != nil {
				return err
			}
			return doMutate(cmd, "PATCH", "/plans/{plan_id}/category_groups/{category_group_id}",
				map[string]string{"category_group_id": id}, jsonBody)
		},
	}
	cmd.Flags().String("category-group-id", "", "Category group ID (required)")
	return cmd
}
