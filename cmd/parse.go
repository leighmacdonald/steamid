package cmd

import (
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"strings"
)

// parseCmd represents the parse command
var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse steam id's from an input file",
	Long: `Parse steam id's from an input file. 

All formats are parsed from the file and duplicates are removed`,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			reader io.Reader
			writer io.Writer
		)
		inputFile := cmd.Flag("input").Value.String()
		outputFile := cmd.Flag("output").Value.String()
		format := strings.ReplaceAll(
			strings.ReplaceAll(cmd.Flag("format").Value.String(), "\\n", "\n"),
			"\\r", "\r")
		idType := strings.ToLower(cmd.Flag("type").Value.String())
		if inputFile != "" {
			i, err := os.Open(inputFile)
			if err != nil {
				log.Fatalf("Failed to open input file (%s): %v", inputFile, err)
			}
			defer func() {
				if err := i.Close(); err != nil {
					log.Printf("Failed to close input file")
				}
			}()
			reader = i
		} else {
			reader = os.Stdin
		}
		if outputFile != "" {
			o, err := os.Create(outputFile)
			if err != nil {
				log.Fatalf("Failed to create output file (%s): %v", outputFile, err)
			}
			defer func() {
				if err := o.Close(); err != nil {
					log.Printf("Failed to close input file")
				}
			}()
			writer = o
		} else {
			writer = os.Stdout
		}

		if err := extra.ParseReader(reader, writer, format, idType); err != nil {
			log.Fatalf(err.Error())
		}
		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)

	parseCmd.Flags().StringP("input", "i", "",
		"Input text file to parse. Uses stdin if not specified.")
	parseCmd.Flags().StringP("output", "o", "",
		"Output results to a file.  Uses stdout if not specified.")
	parseCmd.Flags().StringP("format", "f", "%s\n",
		"Output format to use. Applied to each ID.")
	parseCmd.Flags().StringP("type", "t", "steam64",
		"Output format for steam ids found (steam64, steam, steam3)")
}
