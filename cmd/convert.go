package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/spf13/cobra"
)

func printAllConversions(sid steamid.SteamID) {
	fmt.Printf(`Steam:   %s
Steam3:  %s
Steam32: %d
Steam64: %d`, sid.Steam(false), sid.Steam3(), sid.AccountID, sid.Int64()) //nolint:forbidigo
}

// convertCmd parses and prints out the steam id formats for the input steamid.
var convertCmd = &cobra.Command{ //nolint:exhaustruct,gochecknoglobals
	Use:     "convert",
	Aliases: []string{"c"},
	Args:    cobra.MinimumNArgs(1),
	Short:   "Show steamid conversions",
	Long: `Show steamid conversions.

All formats are parsed from the file and duplicates are removed`,
	Run: func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			sid := steamid.New(arg)
			if !sid.Valid() {
				fmt.Printf("Failed to convert id: %s\n", arg) //nolint:forbidigo
				os.Exit(1)
			}

			idType := ""

			if typeVal := cmd.Flag("format"); typeVal != nil {
				idType = strings.ToLower(typeVal.Value.String())
			}

			switch idType {
			case "":
				printAllConversions(sid)
			case "steam":
				fallthrough
			case "steam2":
				fmt.Printf("%s\n", sid.Steam(false)) //nolint:forbidigo
			case "steam3":
				fmt.Printf("%s\n", sid.Steam3()) //nolint:forbidigo
			case "steam32":
				fmt.Printf("%d\n", sid.AccountID) //nolint:forbidigo
			case "steam64":
				fmt.Printf("%d\n", sid.Int64()) //nolint:forbidigo
			default:
				fmt.Printf("Unknown forma, must be one of steam, steam3, steam32, steam64: %s", idType) //nolint:forbidigo
				os.Exit(1)
			}

		}
		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringP("format", "f", "",
		"Output format to use. Applied to each ID. (steam, steam3, steam32, steam64)")
}
