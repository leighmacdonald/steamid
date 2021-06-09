# steamid

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
![Go](https://github.com/leighmacdonald/steamid/workflows/Go/badge.svg)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/ab0e6cc629b8434ba5dc0803be18bbb4)](https://www.codacy.com/manual/leighmacdonald/steamid?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=leighmacdonald/steamid&amp;utm_campaign=Badge_Grade)
[![Maintainability](https://api.codeclimate.com/v1/badges/3cc77c69032c4e0a917d/maintainability)](https://codeclimate.com/github/leighmacdonald/steamid/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/3cc77c69032c4e0a917d/test_coverage)](https://codeclimate.com/github/leighmacdonald/steamid/test_coverage)
[![Go Report Card](https://goreportcard.com/badge/github.com/leighmacdonald/steamid)](https://goreportcard.com/report/github.com/leighmacdonald/steamid)
[![GoDoc](https://godoc.org/github.com/leighmacdonald/steamid?status.svg)](https://pkg.go.dev/github.com/leighmacdonald/steamid)
![Lines of Code](https://tokei.rs/b1/github/leighmacdonald/steamid)


Simple steamid library that is capable of parsing and converting between all forms of 
steamid's. 

## CLI Usage

The project now also provides an additional CLI for batch conversions.

For compiled binaries for Windows, MacOS and Linux see: [releases](https://github.com/leighmacdonald/steamid/releases).

### Batch parsing

The `parse` command will parse the data for any steamids. It will search for all 
supported ids, removing duplicates. 

Output steamid format can be specified with the `-t` flag. Defaults to steam64.

The `--format` flag applies formatting to the output. The example below (and default), will append a newline
to each steamid found. the `%s` is replaced with the steamid in the format specified.

    $ steamid parse --input ./test_data/log_sup_med_1.log --format "steamid: %s\n" -t steam3
    steamid: [U:1:59956152]
    steamid: [U:1:58024980]
    steamid: [U:1:13379990]
    steamid: [U:1:137497004]
    steamid: [U:1:66374744]
    steamid: [U:1:97609910]
    ...
    $

Piping in via stdin:

    $ cat ./test_data/log_sup_med_1.log | steamid parse -t steam
    STEAM_0:0:42657528
    STEAM_0:0:61488285
    STEAM_0:0:63356089
    STEAM_0:0:527631840
    STEAM_0:0:4807701
    STEAM_0:1:41808234

Note that the results returned are in *no particular order*, so you should sort them
if needed. eg:

    $ steamid parse --input ./test_data/log_sup_med_1.log | sort

Command help:

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

If providing a steam API key with `steamid.SetKey()`, you
can also resolve [vanity](https://partner.steamgames.com/doc/webapi/ISteamUser#ResolveVanityURL) URLs
using steams WebAPI. As well as retrieve player summaries from


## Conversions

It supports all formats of steamid:

- Steam   `STEAM_0:0:86173181`
- Steam3  `[U:1:172346362]`
- Steam32 `172346362`
- Steam64 `76561198132612090`
    
With an API key set, It also supports resolving vanity urls or names like: 

- https://steamcommunity.com/id/SQUIRRELLY
- SQUIRRELLY

## Usage

    $ go get git@github.com:leighmacdonald/steamid.git

```go
package main

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"os"
)

func main() {
	// Parsing vanity profile urls
	// Optional, for resolving vanity names support
	if err := steamid.SetKey("YOUR_STEAM_WEBAPI_KEY"); err != nil {
		fmt.Printf("Invalid steamid: %v", err)
		os.Exit(1)
	}
	resolvedSID64, err := steamid.ResolveVanity(context.Background(), "https://steamcommunity.com/id/SQUIRRELLY")
	if err != nil {
		fmt.Printf("Could not resolve: %v", err)
	}
	fmt.Printf("Resolved to: %d\n", resolvedSID64)

	// Normal conversions like these do not require a key to be set
	sid64, errConv := steamid.StringToSID64("76561197961279983")
	if errConv != nil {
		fmt.Printf("Could not convert string: %v", errConv)
	}
	if sid64 != resolvedSID64 {
		fmt.Printf("They dont match!")
	}
	fmt.Printf("Steam64: %d\n", sid64)
	fmt.Printf("Steam32: %d\n", steamid.SID64ToSID32(sid64))
	fmt.Printf("Steam3: %s\n", steamid.SID64ToSID3(sid64))
	fmt.Printf("Steam: %s\n", steamid.SID64ToSID(sid64))
}

```

## Extra Functions

The extra package also includes some helpful functions that are used to fetch & parse common data formats related
to steam ids. These are not directly related to steamid conversions, but are a common use case.

- Retrieve the [GetPlayerSummaries](https://developer.valvesoftware.com/wiki/Steam_Web_API#GetPlayerSummaries_.28v0002.29) 
Steam WebAPI into a `extra.[]PlayerSummary` structure: `PlayerSummaries(ctx context.Context, steamIDs []steamid.SID64) ([]PlayerSummary, error)`
- Parsing all status console command output: `extra.ParseStatus(status string, full bool) (Status, error)`
- Parse just the status console steamids: `extra.SIDSFromStatus(text string) []steamid.SID64` 
- Parse all steamids from a input `io.Reader` into a `io.Writer` using a custom format. This is the 
programmatic way to do what the cli `parse` command does: `extra.ParseReader(input io.Reader, output io.Writer, format string, idType string) error`

## Docs

Here you can find the full [documentation](https://pkg.go.dev/github.com/leighmacdonald/steamid).
