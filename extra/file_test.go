package extra_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/leighmacdonald/steamid/v4/extra"

	"github.com/stretchr/testify/require"
)

func TestParseInput(t *testing.T) {
	t.Parallel()

	testBody := `# userid name                uniqueid            connected ping loss state
#      2 "WolfXine"          [U:1:166779318]     15:22       85    0 active
#      3 "mdaniels5746"      [U:1:361821288]     15:22       87    0 active
#     28 "KRGonzales"        [U:1:875620767]     00:29       76   10 active
#      4 "juan.martinez2009" [U:1:79002518]      15:22       72    0 active
[U:1:172346362]STEAM_0:0:86173182[U:1:172346342]
STEAM_0:0:86173181
76561198132612090

76561198084134025
`

	ids := extra.FindReaderSteamIDs(strings.NewReader(testBody))
	require.Len(t, ids, 8) // 2 duplicated
}

func TestParseReader(t *testing.T) {
	testBody := `# userid name                uniqueid            connected ping loss state
#      2 "WolfXine"          [U:1:166779318]     15:22       85    0 active
#      3 "mdaniels5746"      [U:1:361821288]     15:22       87    0 active
#     28 "KRGonzales"        [U:1:875620767]     00:29       76   10 active
#      4 "juan.martinez2009" [U:1:79002518]      15:22       72    0 active

[U:1:172346362]STEAM_0:0:86173182[U:1:172346342]
STEAM_0:0:86173181
76561198132612090
76561198084134025

`
	for format, expected := range map[string]string{
		"steam64": "-76561198127045046-\n-76561198322087016-\n-76561198835886495-\n-76561198039268246-\n" +
			"-76561198132612092-\n-76561198132612090-\n-76561198132612070-\n-76561198084134025-\n",
		"steam3": "-[U:1:166779318]-\n-[U:1:361821288]-\n-[U:1:875620767]-\n-[U:1:79002518]-\n" +
			"-[U:1:172346364]-\n-[U:1:172346362]-\n-[U:1:172346342]-\n-[U:1:123868297]-\n",
		"steam": "-STEAM_0:0:83389659-\n-STEAM_0:0:180910644-\n-STEAM_0:1:437810383-\n-STEAM_0:0:39501259-\n" +
			"-STEAM_0:0:86173182-\n-STEAM_0:0:86173181-\n-STEAM_0:0:86173171-\n-STEAM_0:1:61934148-\n",
		"steam32": "-166779318-\n-361821288-\n-875620767-\n-79002518-\n-172346364-\n-172346362-\n-172346342-\n-123868297-\n",
	} {
		var buf64 bytes.Buffer
		require.NoError(t, extra.ParseReader(strings.NewReader(testBody), &buf64, "-%s-\n", format))
		require.Equalf(t, expected, buf64.String(), "Failed to generate: %s", format)
	}
}
