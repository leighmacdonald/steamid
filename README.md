# steamid
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) 
[![TravisBuild](https://api.travis-ci.com/leighmacdonald/steamid.svg?branch=master)](https://api.travis-ci.com/leighmacdonald/steamid.svg?branch=master)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/ab0e6cc629b8434ba5dc0803be18bbb4)](https://www.codacy.com/manual/leighmacdonald/steamid?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=leighmacdonald/steamid&amp;utm_campaign=Badge_Grade)
[![Maintainability](https://api.codeclimate.com/v1/badges/3cc77c69032c4e0a917d/maintainability)](https://codeclimate.com/github/leighmacdonald/steamid/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/3cc77c69032c4e0a917d/test_coverage)](https://codeclimate.com/github/leighmacdonald/steamid/test_coverage)
[![Go Report Card](https://goreportcard.com/badge/github.com/leighmacdonald/steamid)](https://goreportcard.com/report/github.com/leighmacdonald/steamid)
[![GoDoc](https://godoc.org/github.com/leighmacdonald/steamid?status.svg)](https://pkg.go.dev/github.com/leighmacdonald/steamid)
![Lines of Code](https://tokei.rs/b1/github/leighmacdonald/steamid)

Simple steamid library that is capable of parsing and converting between all forms of 
steamid's. 

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
    "github.com/leighmacdonald/steamid"
)
    
func main() {
    steamid.SetKey("YOUR_STEAM_WEBAPI_KEY") // Optional, for resolving vanity names support
    resolvedSID64, err := steamid.ResolveVanity(context.Background(), "https://steamcommunity.com/id/SQUIRRELLY")
    if err != nil {
        fmt.Printf("Could not resolve: %v", err)
    }
    // Normal conversions like these do not require a key to be set
    sid64, err := steamid.StringToSID64("76561197961279983")
    if err != nil {
        fmt.Printf("Could not convert string: %v", err)
    }
    if sid64 != resolvedSID64 {
        fmt.Printf("They dont match!")
    }
    fmt.Printf("Steam64: %d\n", sid64)
    fmt.Printf("Steam32: %d\n", steamid.SID64ToSID32(sid64))
    fmt.Printf("Steam3: %s\n", steamid.SID64ToSID3(sid64))
}

```

## Docs

Here you can find the full [documentation](https://pkg.go.dev/github.com/leighmacdonald/steamid).
