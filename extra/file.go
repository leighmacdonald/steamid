package extra

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/leighmacdonald/steamid/v3/steamid"
)

var (
	ErrIDType = errors.New("invalid sid type")

	ErrReadInput = errors.New("error while reading input")

	ErrWrite = errors.New("failed to write to output file")

	ErrFlush = errors.New("failed to flush contents")
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
	case "steam64":
	default:
		return fmt.Errorf("%w: %s", ErrIDType, idType)
	}

	writer := bufio.NewWriter(output)
	reader := bufio.NewScanner(input)

	var lines []string

	for reader.Scan() {
		lines = append(lines, reader.Text())
	}

	if err := reader.Err(); err != nil {
		return errors.Join(err, ErrReadInput)
	}

	for _, id := range steamid.ParseString(strings.Join(lines, "")) {
		value := ""

		switch idType {
		case "steam64":
			value = id.String()
		case "steam3":
			value = string(id.Steam3())
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
