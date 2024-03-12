package cmd

import (
	"io"
	"log"
	"os"
	"strings"

	"github.com/leighmacdonald/steamid/v4/extra"
	"github.com/spf13/cobra"
)

// parseCmd represents the parse command.
var parseCmd = &cobra.Command{ //nolint:exhaustruct,gochecknoglobals
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
		outputFilePath := cmd.Flag("output").Value.String()
		format := strings.ReplaceAll(
			strings.ReplaceAll(cmd.Flag("format").Value.String(), "\\n", "\n"),
			"\\r", "\r")
		idType := strings.ToLower(cmd.Flag("type").Value.String())
		if inputFile != "" {
			openedInputFile, errOpen := os.Open(inputFile)
			if errOpen != nil {
				log.Fatalf("Failed to open input file (%s): %v", inputFile, errOpen)
			}
			defer func() {
				if err := openedInputFile.Close(); err != nil {
					log.Printf("Failed to close input file")
				}
			}()
			reader = openedInputFile
		} else {
			reader = os.Stdin
		}
		if outputFilePath != "" {
			outFile, err := os.Create(outputFilePath)
			if err != nil {
				log.Fatalf("Failed to create output file (%s): %v", outputFilePath, err)
			}
			defer func() {
				if err := outFile.Close(); err != nil {
					log.Printf("Failed to close input file")
				}
			}()
			writer = outFile
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
