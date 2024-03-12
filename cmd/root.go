// Package cmd implements the CLI interface for steamid
package cmd

import (
	"fmt"
	"os"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{ //nolint:exhaustruct,gochecknoglobals
	Use:   "steamid",
	Short: "A library and CLI app to convert between steam id formats",
	Long:  `A library and CLI app to convert between steam id formats`,
	//	Run: func(cmd *cobra.Command, args []string) { },
	Version: fmt.Sprintf("%s - %s", steamid.BuildVersion, steamid.BuildDate),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
