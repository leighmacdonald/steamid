package extra

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
)

// ParseReader attempt to find all types of steam ids in the data stream provided by the
// input reader. It will write the output of what it finds to the output writer applying the
// formatting strings to each value. The formatting string takes the same formatting as the
// standards fmt.SprintF() and expects one %s token.
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
		return errors.Errorf("invalid id type: %s", idType)
	}
	writer := bufio.NewWriter(output)
	reader := bufio.NewScanner(input)
	var lines []string
	for reader.Scan() {
		lines = append(lines, reader.Text())
	}
	if err := reader.Err(); err != nil {
		return errors.Errorf("Error reading input: %v", err)
	}
	ids64 := steamid.ParseString(strings.Join(lines, ""))
	for _, id := range ids64 {
		v := ""
		switch idType {
		case "steam64":
			v = id.String()
		case "steam32":
			v = strconv.FormatInt(int64(steamid.SID64ToSID32(id)), 10)
		case "steam3":
			v = string(steamid.SID64ToSID3(id))
		case "steam":
			v = string(steamid.SID64ToSID(id))
		}
		_, err := writer.WriteString(fmt.Sprintf(format, v))
		if err != nil {
			return errors.Wrapf(err, "Error writing id to output")
		}
		if err := writer.Flush(); err != nil {
			return errors.Wrapf(err, "Failed to flush remaining data")
		}
	}
	return nil
}
