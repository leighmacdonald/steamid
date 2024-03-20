package extra

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrIDType = errors.New("invalid sid type")
	ErrWrite  = errors.New("failed to write to output file")
	ErrFlush  = errors.New("failed to flush contents")
)

// ParseReader attempt to find all types of steam ids in the data stream provided by the
// input reader. It will write the output of what it finds to the output writer applying the
// formatting strings to each value. The formatting string takes the same formatting as the
// standard fmt.SprintF() and expects one %s token.
//
// A formatting example to place each steam id on a newline: "%s\n"
//
// idType specifies what output id format to use when writing: steam, steam3, steam32, steam64 are
// the valid choices.
func ParseReader(input io.Reader, output io.Writer, format string, idType string) error {
	switch idType {
	case "steam":
	case "steam3":
	case "steam32":
	case "steam64":
	default:
		return fmt.Errorf("%w: %s", ErrIDType, idType)
	}

	writer := bufio.NewWriter(output)

	for _, id := range FindReaderSteamIDs(input) {
		value := ""

		switch idType {
		case "steam64":
			value = id.String()
		case "steam3":
			value = string(id.Steam3())
		case "steam32":
			value = fmt.Sprintf("%d", id.AccountID)
		case "steam":
			value = string(id.Steam(false))
		}

		_, errWrite := writer.WriteString(fmt.Sprintf(format, value))
		if errWrite != nil {
			return errors.Join(errWrite, ErrWrite)
		}

		if errFlush := writer.Flush(); errFlush != nil {
			return errors.Join(errFlush, ErrFlush)
		}
	}

	return nil
}

// FindReaderSteamIDs attempts to parse any strings of any known format within the body to a common SID64 format.
func FindReaderSteamIDs(reader io.Reader) []steamid.SteamID {
	var (
		scanner  = bufio.NewScanner(reader)
		freSID   = regexp.MustCompile(`STEAM_0:[01]:[0-9][0-9]{0,8}`)
		freSID64 = regexp.MustCompile(`7656119\d{10}`)
		freSID3  = regexp.MustCompile(`\[U:1:\d+]`)
		// Store only unique entries
		found []steamid.SteamID
	)

	for scanner.Scan() {
		line := scanner.Text()
		if matches := freSID.FindAllStringSubmatch(line, -1); matches != nil {
			for _, i := range matches {
				sid := steamid.New(i[0])
				if !sid.Valid() {
					continue
				}
				found = append(found, sid)
			}
		}
		if matches := freSID64.FindAllStringSubmatch(line, -1); matches != nil {
			for _, i := range matches {
				sid := steamid.New(i[0])
				if !sid.Valid() {
					continue
				}
				found = append(found, sid)
			}
		}
		if matches := freSID3.FindAllStringSubmatch(line, -1); matches != nil {
			for _, i := range matches {
				sid := steamid.New(i[0])
				if !sid.Valid() {
					continue
				}
				found = append(found, sid)
			}
		}
	}

	var uniq []steamid.SteamID
	for _, foundID := range found {
		if !slices.ContainsFunc(uniq, func(sid steamid.SteamID) bool {
			return foundID.Int64() == sid.Int64()
		}) {
			uniq = append(uniq, foundID)
		}
	}

	return uniq
}
