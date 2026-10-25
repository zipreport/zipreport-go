package main

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("zipreport version %s\n", currentVersion())
	},
}

// resolveVersion prefers the ldflags value, then the module version Go
// recorded at build time, and falls back to "dev" when there is none.
func resolveVersion(ldflags string, info *debug.BuildInfo) string {
	if ldflags != "" {
		return ldflags
	}
	if info != nil && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "dev"
}

// currentVersion is the version this binary reports.
func currentVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		info = nil
	}
	return resolveVersion(version, info)
}
