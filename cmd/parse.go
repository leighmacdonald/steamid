package cmd

import (
	"bufio"
	"fmt"
	"github.com/leighmacdonald/steamid/steamid"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strings"
)

var testBody = `version : 4620606/24 4620606 secure
hostname: Valve Matchmaking Server (Washington mwh-1/srcds135 #48)
udp/ip  : 192.69.97.58:27062  (public ip: 192.69.97.58)
steamid : [A:1:729372672:10372] (90116540677576704)
map     : koth_suijin at: 0 x, 0 y, 0 z
account : not logged in  (No account specified)
tags    : cp,hidden,increased_maxplayers,valve
players : 24 humans, 0 bots (32 max)
edicts  : 731 used of 2048 max
# userid name                uniqueid            connected ping loss state
#      2 "WolfXine"          [U:1:166779318]     15:22       85    0 active
#      3 "mdaniels5746"      [U:1:361821288]     15:22       87    0 active
#     28 "KRGonzales"        [U:1:875620767]     00:29       76   10 active
#      4 "juan.martinez2009" [U:1:79002518]      15:22       72    0 active
#      9 "Luuá¸°e"           [U:1:123675776]     15:18      109    0 active
#      5 "[LBJ] â™› King James â™›" [U:1:87772789] 15:22     76    0 active
#     10 "elirobot"          [U:1:167562538]     15:18       91    0 active
#      6 "guy (actual human feces)" [U:1:416855641] 15:22    83    0 active
#     26 "=/TFA\= CosmicTat" [U:1:163325254]     00:38       94    0 active
#      7 "alterego312"       [U:1:242237960]     15:22      128    0 active
#     12 "KcTheCray"         [U:1:332143895]     15:17       90    0 active
#      8 "Fag Bag McGee | Trade.tf" [U:1:861259628] 15:22   127    0 active
#     13 "Prototype x1-5150" [U:1:339990071]     15:17       77    0 active
#     14 "VAVI"              [U:1:122890196]     15:09      132    0 active
#     15 "Mecha Engineer Alfredo" [U:1:196165302] 15:06     132    0 active
#     16 "Ceebee324"         [U:1:132135410]     14:45      102    0 active
#     19 "Lil Dave"          [U:1:123147588]     14:39       87    0 active
#     22 "Stede Bonnet the pirate" [U:1:206922652] 10:37    165    0 active
#     20 "hard aim pootis serbia" [U:1:49974197] 14:13       84    0 active
#     18 "Enderz"            [U:1:202535707]     14:41       83    0 active
#     23 "WAFFLEDUDE"        [U:1:878783526]     10:33      128    0 active
#     24 "smokehousesteve"   [U:1:130361378]     09:54      128    0 active
#     29 "à¸¸"               [U:1:123868297]     00:24       59    0 active
#     27 "Cyndaquil"         [U:1:198198697]     00:31      131    0 active`

// parseCmd represents the parse command
var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse steam id's from an input file",
	Long: `Parse steam id's from an input file. 

All formats are parsed from the file and duplicates are removed`,
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := cmd.Flag("input").Value.String()
		outputFile := cmd.Flag("output").Value.String()
		format := strings.ReplaceAll(
			strings.ReplaceAll(cmd.Flag("format").Value.String(), "\\n", "\n"),
			"\\r", "\r")
		idType := strings.ToLower(cmd.Flag("type").Value.String())
		switch idType {
		case "steam64":
		case "steam":
		case "steam3":
		default:
			log.Fatalf("invalid id type: %s", idType)
		}
		var reader *bufio.Scanner
		var writer *bufio.Writer
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
			reader = bufio.NewScanner(i)
		} else {
			reader = bufio.NewScanner(os.Stdin)
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
			writer = bufio.NewWriter(o)
		} else {
			writer = bufio.NewWriter(os.Stdout)
		}
		var lines []string
		for reader.Scan() {
			lines = append(lines, reader.Text())
		}
		if err := reader.Err(); err != nil {
			log.Fatalf("Error reading input: %v", err)
		}
		ids64 := steamid.ParseString(strings.Join(lines, ""))
		for _, id := range ids64 {
			v := ""
			switch idType {
			case "steam64":
				v = id.String()
			case "steam3":
				v = string(steamid.SID64ToSID3(id))
			case "steam":
				v = string(steamid.SID64ToSID(id))
			}
			_, err := writer.WriteString(fmt.Sprintf(format, v))
			if err != nil {
				log.Fatalf("Error writing id to output: %v", err)
			}
			if err := writer.Flush(); err != nil {
				log.Printf("Failed to flush remaining data: %v", err)
			}
		}
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
