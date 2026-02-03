// gen-docs generates CLI reference documentation from the Cobra command tree.
//
// Usage:
//
//	go run ./cmd/gen-docs [--out DIR]
//
// By default, it writes Starlight-compatible markdown files to docs/src/content/docs/reference/.
// Each command gets its own page with frontmatter, flags, and examples.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra/doc"
	"github.com/triptechtravel/clickup-cli/internal/iostreams"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/root"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

func main() {
	outDir := "docs/src/content/docs/reference"
	if len(os.Args) > 2 && os.Args[1] == "--out" {
		outDir = os.Args[2]
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("failed to create output directory: %v", err)
	}

	// Build the command tree. Auth is not needed for doc generation.
	ios := iostreams.System()
	f := cmdutil.NewFactory(ios)
	rootCmd := root.NewCmdRoot(f)
	rootCmd.DisableAutoGenTag = true

	// Prepend Starlight frontmatter to each generated file.
	prepender := func(filename string) string {
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, filepath.Ext(name))

		// Convert "clickup_task_view" to "clickup task view"
		title := strings.ReplaceAll(base, "_", " ")

		return fmt.Sprintf(`---
title: "%s"
description: "Auto-generated reference for %s"
---

`, title, title)
	}

	// Link handler for cross-references between command pages.
	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, filepath.Ext(name))
		return "/clickup-cli/reference/" + strings.ToLower(base) + "/"
	}

	if err := doc.GenMarkdownTreeCustom(rootCmd, outDir, prepender, linkHandler); err != nil {
		log.Fatalf("failed to generate docs: %v", err)
	}

	// Count generated files.
	entries, _ := os.ReadDir(outDir)
	fmt.Printf("Generated %d reference pages in %s\n", len(entries), outDir)
}
