package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/glebmish/ynab-cli/internal/api"
	"github.com/glebmish/ynab-cli/internal/cliexit"
	"github.com/glebmish/ynab-cli/internal/format"
	"github.com/glebmish/ynab-cli/internal/validate"
	"github.com/spf13/cobra"
)

// validationErr wraps a validate-package error so main.go can exit with code 3.
func validationErr(err error) error {
	if err == nil {
		return nil
	}
	return &cliexit.ValidationError{Err: err}
}

// validateMonth allows "current" or YYYY-MM-DD.
func validateMonth(value string) error {
	if value == "current" {
		return nil
	}
	return validate.DateParam("month", value)
}

func fmtOpts(cmd *cobra.Command) format.Options {
	f, _ := cmd.Flags().GetString("format")
	fields, _ := cmd.Flags().GetString("fields")
	flatten, _ := cmd.Flags().GetBool("flatten-splits")
	return format.FormatFromFlags(f, fields, flatten)
}

// mergeParams overlays the persistent --params JSON object onto the
// command-built params map. Caller-set entries win on key collision so
// commands stay authoritative for their core inputs; --params fills the gaps
// the command didn't expose as named flags.
func mergeParams(cmd *cobra.Command, base map[string]string) (map[string]string, error) {
	merged := map[string]string{}
	raw, _ := cmd.Flags().GetString("params")
	if raw != "" {
		if err := validate.JSONBody(raw); err != nil {
			return nil, err
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return nil, validationErr(fmt.Errorf("--params: %w", err))
		}
		for k, v := range m {
			merged[k] = fmt.Sprint(v)
		}
	}
	for k, v := range base {
		merged[k] = v
	}
	return merged, nil
}

// requireString reads a required string flag or returns a descriptive error.
func requireString(cmd *cobra.Command, name string) (string, error) {
	v, _ := cmd.Flags().GetString(name)
	if v == "" {
		return "", validationErr(fmt.Errorf("--%s is required", name))
	}
	return v, nil
}

// requireJSON reads --json and validates it.
func requireJSON(cmd *cobra.Command) (string, error) {
	body, err := requireString(cmd, "json")
	if err != nil {
		return "", err
	}
	if err := validate.JSONBody(body); err != nil {
		return "", err
	}
	return body, nil
}

func doGet(cmd *cobra.Command, path string, params map[string]string) error {
	merged, err := mergeParams(cmd, params)
	if err != nil {
		return err
	}
	c := api.FromContext(cmd.Context())

	if dr, _ := cmd.Flags().GetBool("dry-run"); dr {
		fmt.Println(c.DryRun("GET", path, merged, nil))
		return nil
	}

	resp, err := c.Do("GET", path, merged, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	return format.Write(os.Stdout, body, fmtOpts(cmd))
}

func doMutate(cmd *cobra.Command, method, path string, params map[string]string, jsonFlag string) error {
	merged, err := mergeParams(cmd, params)
	if err != nil {
		return err
	}
	c := api.FromContext(cmd.Context())

	var body []byte
	if jsonFlag != "" {
		if err := validate.JSONBody(jsonFlag); err != nil {
			return err
		}
		body = []byte(jsonFlag)
	}

	if dr, _ := cmd.Flags().GetBool("dry-run"); dr {
		fmt.Println(c.DryRun(method, path, merged, body))
		return nil
	}

	resp, err := c.Do(method, path, merged, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if len(respBody) == 0 {
		return nil
	}
	return format.Write(os.Stdout, respBody, fmtOpts(cmd))
}

func doDelete(cmd *cobra.Command, path string, params map[string]string, resource, id string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if !dryRun {
		if err := confirmDelete(cmd, resource, id); err != nil {
			return err
		}
	}

	merged, err := mergeParams(cmd, params)
	if err != nil {
		return err
	}
	c := api.FromContext(cmd.Context())

	if dryRun {
		fmt.Println(c.DryRun("DELETE", path, merged, nil))
		return nil
	}

	resp, err := c.Do("DELETE", path, merged, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if len(body) > 0 {
		return format.Write(os.Stdout, body, fmtOpts(cmd))
	}
	return nil
}
