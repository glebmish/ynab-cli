package cmd

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed skills
var skillsFS embed.FS

var installSkillsCmd = &cobra.Command{
	Use:   "install-skills",
	Short: "Install AI agent skill files",
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
	installSkillsCmd.Flags().String("output-dir", "", "Output directory (skip interactive prompt)")
	rootCmd.AddCommand(installSkillsCmd)
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

	err := fs.WalkDir(skillsFS, "skills", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath := strings.TrimPrefix(path, "skills/")
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
