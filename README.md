# steamid


Simple steamid library that is capable of parsing and converting between all forms of 
steamid's. 

## CLI Usage

The project now also provides an additional CLI for batch conversions.

For compiled binaries for Windows, MacOS and Linux see: [releases](https://github.com/leighmacdonald/steamid/releases).

### Batch parsing

The `parse` command will parse the data for any steamids. It will search for all 
supported ids, removing duplicates. 

Output steamid format can be specified with the `-t` flag. Defaults to steam64.

The `--format` flag applies formatting to the output. The exaple below (and default), will append a newline
to each steamid found. the `%s` is replaced with the steamid in the format specified.

`steamid parse --input input.txt --format "%s\n" -t steam3 -h`


```
Parse steam id's from an input file.

All formats are parsed from the file and duplicates are removed

Usage:
  steamid parse [flags]

Flags:
  -f, --format string   Output format to use. Applied to each ID. (default "%s\n")
  -h, --help            help for parse
  -i, --input string    Input text file to parse. Uses stdin if not specified.
  -o, --output string   Output results to a file.  Uses stdout if not specified.
  -t, --type string     Output format for steam ids found (steam64, steam, steam3) (default "steam64")


```

## Library Usage

To see how to use this as a library, please check the 
generated [docs](https://pkg.go.dev/github.com/leighmacdonald/steamid)

## Vanity URL

If providing a steam API key with steamid.SetKey(), you
can also resolve [vanity](https://partner.steamgames.com/doc/webapi/ISteamUser#ResolveVanityURL) URLs
using steams WebAPI. 