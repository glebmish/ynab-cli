// internal/cmd/skills.go — `skills` command group: list, get, install.
//
// `skills list` and `skills get` give agents a runtime path to bundled skill
// content with no on-disk install required. `skills install` is the existing
// disk-install UX, moved under the group.
package cmd

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebmish/ynab-cli/internal/cliexit"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

//go:embed skills
var skillsFS embed.FS

const skillsRoot = "skills"

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Browse and install bundled agent skills",
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List bundled agent skills (text; --format json emits full frontmatter)",
	RunE:  runSkillsList,
}

var skillsGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Print one bundled skill (raw body by default; --format json emits a full envelope)",
	Args:  cobra.ExactArgs(1),
	RunE:  runSkillsGet,
}

var skillsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install bundled skills to disk",
	Long: `Install skill files that help AI agents (Claude Code, etc.) use the ynab CLI effectively.

Skills are structured Markdown files that teach agents about available commands,
required flags, safety rules, and multi-step budgeting workflows.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, _ := cmd.Flags().GetString("output-dir")
		if outputDir != "" {
			return installSkills(outputDir)
		}
		return installSkillsInteractive()
	},
}

func init() {
	skillsInstallCmd.Flags().String("output-dir", "", "Output directory (skip interactive prompt)")
	skillsCmd.AddCommand(skillsListCmd, skillsGetCmd, skillsInstallCmd)
	rootCmd.AddCommand(skillsCmd)
}

// jsonFormatRequested reports true when the user explicitly passed --format json.
// The persistent --format flag defaults to "json" for API response formatting,
// but the skills commands default to text/raw; opt into JSON only on explicit
// request.
func jsonFormatRequested(cmd *cobra.Command) bool {
	if !cmd.Flags().Changed("format") {
		return false
	}
	format, _ := cmd.Flags().GetString("format")
	return format == "json"
}

func runSkillsList(cmd *cobra.Command, args []string) error {
	entries, err := listSkillsMeta()
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	if jsonFormatRequested(cmd) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}
	for _, e := range entries {
		name, _ := e["name"].(string)
		desc, _ := e["description"].(string)
		fmt.Fprintf(w, "%-30s  %s\n", name, desc)
	}
	return nil
}

func runSkillsGet(cmd *cobra.Command, args []string) error {
	name := args[0]
	skillDir := filepath.Join(skillsRoot, name)
	if _, err := skillsFS.ReadDir(skillDir); err != nil {
		return &cliexit.ValidationError{Err: fmt.Errorf("unknown skill %q. Run `ynab skills list` to see available skills", name)}
	}
	skillMDPath := filepath.Join(skillDir, "SKILL.md")
	content, err := skillsFS.ReadFile(skillMDPath)
	if err != nil {
		return &cliexit.DiscoveryError{Err: fmt.Errorf("reading %s: %w", skillMDPath, err)}
	}

	w := cmd.OutOrStdout()
	if jsonFormatRequested(cmd) {
		return writeSkillEnvelope(w, name, skillDir, content)
	}

	_, body := splitFrontmatter(content)
	if len(body) == 0 {
		body = content
	}
	_, err = w.Write(body)
	return err
}

// listSkillsMeta walks the embedded skills directory and returns one
// frontmatter map per skill (subdirectories that contain a SKILL.md).
// Frontmatter is passed through as map[string]any — whatever keys the
// SKILL.md declares are emitted, so the contract stays stable as skill
// authors add fields.
func listSkillsMeta() ([]map[string]any, error) {
	dirEntries, err := skillsFS.ReadDir(skillsRoot)
	if err != nil {
		return nil, &cliexit.DiscoveryError{Err: fmt.Errorf("reading bundled skills: %w", err)}
	}
	var out []map[string]any
	for _, e := range dirEntries {
		if !e.IsDir() {
			continue
		}
		mdPath := filepath.Join(skillsRoot, e.Name(), "SKILL.md")
		content, err := skillsFS.ReadFile(mdPath)
		if err != nil {
			continue
		}
		meta, err := parseSkillFrontmatter(mdPath, content)
		if err != nil {
			return nil, err
		}
		out = append(out, meta)
	}
	return out, nil
}

func writeSkillEnvelope(w io.Writer, name, skillDir string, primaryContent []byte) error {
	meta, err := parseSkillFrontmatter(filepath.Join(skillDir, "SKILL.md"), primaryContent)
	if err != nil {
		return err
	}
	var files []map[string]string
	err = fs.WalkDir(skillsFS, skillDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, rerr := skillsFS.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		rel := strings.TrimPrefix(path, skillDir+"/")
		files = append(files, map[string]string{"path": rel, "content": string(data)})
		return nil
	})
	if err != nil {
		return &cliexit.DiscoveryError{Err: err}
	}
	desc, _ := meta["description"].(string)
	env := map[string]any{
		"name":        name,
		"description": desc,
		"files":       files,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}

// splitFrontmatter splits SKILL.md content into (frontmatterYAML, body).
// Returns (nil, content) if the file doesn't start with a frontmatter block.
func splitFrontmatter(content []byte) (frontmatter, body []byte) {
	var rest []byte
	switch {
	case bytes.HasPrefix(content, []byte("---\n")):
		rest = content[4:]
	case bytes.HasPrefix(content, []byte("---\r\n")):
		rest = content[5:]
	default:
		return nil, content
	}
	end := bytes.Index(rest, []byte("\n---\n"))
	delimLen := len("\n---\n")
	if end < 0 {
		end = bytes.Index(rest, []byte("\n---\r\n"))
		delimLen = len("\n---\r\n")
	}
	if end < 0 {
		return nil, content
	}
	fm := rest[:end]
	body = rest[end+delimLen:]
	body = bytes.TrimLeft(body, "\n")
	return fm, body
}

func parseSkillFrontmatter(path string, content []byte) (map[string]any, error) {
	fm, _ := splitFrontmatter(content)
	if len(fm) == 0 {
		return nil, &cliexit.DiscoveryError{Err: fmt.Errorf("%s: missing frontmatter", path)}
	}
	var meta map[string]any
	if err := yaml.Unmarshal(fm, &meta); err != nil {
		return nil, &cliexit.DiscoveryError{Err: fmt.Errorf("%s: parsing frontmatter: %w", path, err)}
	}
	return meta, nil
}

func installSkillsInteractive() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprintln(os.Stderr, "Where should skills be installed?")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  Scope:")
	fmt.Fprintln(os.Stderr, "    1) Project — available in this directory only")
	fmt.Fprintln(os.Stderr, "    2) User    — available in all projects")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprint(os.Stderr, "Choose scope [1]: ")
	scopeInput, _ := reader.ReadString('\n')
	scopeInput = strings.TrimSpace(scopeInput)
	if scopeInput == "" {
		scopeInput = "1"
	}

	var baseDirs []string
	var scopeLabel string

	switch scopeInput {
	case "1":
		scopeLabel = "project"
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		baseDirs = []string{
			filepath.Join(cwd, ".claude", "skills"),
			filepath.Join(cwd, ".agents", "skills"),
		}
	case "2":
		scopeLabel = "user"
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}
		baseDirs = []string{
			filepath.Join(home, ".claude", "skills"),
			filepath.Join(home, ".agents", "skills"),
		}
	default:
		return fmt.Errorf("invalid scope: %q (expected 1 or 2)", scopeInput)
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "  Directory (%s):\n", scopeLabel)
	for i, dir := range baseDirs {
		fmt.Fprintf(os.Stderr, "    %d) %s\n", i+1, dir)
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprint(os.Stderr, "Choose directory [1]: ")
	dirInput, _ := reader.ReadString('\n')
	dirInput = strings.TrimSpace(dirInput)
	if dirInput == "" {
		dirInput = "1"
	}

	var idx int
	switch dirInput {
	case "1":
		idx = 0
	case "2":
		idx = 1
	default:
		return fmt.Errorf("invalid choice: %q (expected 1 or 2)", dirInput)
	}

	return installSkills(baseDirs[idx])
}

func installSkills(outputDir string) error {
	var written int

	err := fs.WalkDir(skillsFS, skillsRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath := strings.TrimPrefix(path, skillsRoot+"/")
		if relPath == "" {
			return nil
		}

		destPath := filepath.Join(outputDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := skillsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", destPath, err)
		}

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", destPath, err)
		}

		fmt.Fprintf(os.Stderr, "  %s\n", destPath)
		written++
		return nil
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "\nInstalled %d skill files to %s\n", written, outputDir)
	return nil
}
