package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zipreport/zipreport-go/pkg/report"
)

var buildCmd = &cobra.Command{
	Use:   "build <source_dir> <output.zpt>",
	Short: "Build a .zpt file from a template directory",
	Long: `Build assembles a .zpt file from a template directory.

The source directory must contain at minimum:
  - manifest.json: Report metadata
  - index.html: Main template

Example:
  zipreport build ./my-template output.zpt
  zipreport build -s ./my-template output.zpt  # Follow symlinks`,
	Args: cobra.ExactArgs(2),
	RunE: runBuild,
}

var followSymlinks bool

func init() {
	buildCmd.Flags().BoolVarP(&followSymlinks, "symlinks", "s", false, "Follow symlinks when building")
}

func runBuild(cmd *cobra.Command, args []string) error {
	srcDir := args[0]
	destFile := args[1]

	opts := report.DefaultBuildOptions()
	opts.FollowSymlinks = followSymlinks

	fmt.Printf("Building %s from %s...\n", destFile, srcDir)

	result := report.Build(srcDir, destFile, opts)
	if !result.Success() {
		return fmt.Errorf("build failed: %s", result.Error())
	}

	fmt.Printf("Successfully built %s\n", destFile)
	return nil
}
