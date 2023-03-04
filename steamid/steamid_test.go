package steamid

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestRandSID64(t *testing.T) {
	sid := RandSID64()
	require.True(t, sid.Valid())
}

func TestSID64FromString(t *testing.T) {
	v, err := SID64FromString("76561198132612090")
	require.NoError(t, err)
	require.Equal(t, SID64(76561198132612090), v)
	v2, err := SID64FromString("asdf")
	require.Error(t, err)
	require.Equal(t, SID64(0), v2)
	v3, err := SID64FromString("")
	require.Error(t, err)
	require.Equal(t, SID64(0), v3)
}

func TestGIDFromString(t *testing.T) {
	g0, err := GIDFromString("103582791441572968")
	require.NoError(t, err)
	require.Equal(t, GID(103582791441572968), g0)
	g1, err := GIDFromString("asdf")
	require.Error(t, err)
	require.Equal(t, GID(0), g1)
	g2, err := GIDFromString("")
	require.Error(t, err)
	require.Equal(t, GID(0), g2)
}

func TestParseString(t *testing.T) {
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
	ids := ParseString(testBody)
	require.Len(t, ids, 8) // 2 duplicated
}

func TestConversions(t *testing.T) {
	// id := 76561197970669109
	require.Equal(t, SID64ToSID3(76561199127271263), SID3("[U:1:1167005535]"))
	require.Equal(t, SID3ToSID32("[U:1:172346362]"), SID32(172346362))
	require.Equal(t, SID3ToSID64("[U:1:172346362]"), SID64(76561198132612090))
	require.Equal(t, SID3ToSID("[U:1:172346362]"), SID("STEAM_0:0:86173181"))
	require.Equal(t, SID32ToSID3(172346362), SID3("[U:1:172346362]"))
	require.Equal(t, SID32ToSID64(172346362), SID64(76561198132612090))
	require.Equal(t, SID32ToSID(172346362), SID("STEAM_0:0:86173181"))
	require.Equal(t, SID64ToSID3(76561198132612090), SID3("[U:1:172346362]"))
	require.Equal(t, SID64ToSID32(76561198132612090), SID32(172346362))
	require.Equal(t, SID64ToSID(76561198132612090), SID("STEAM_0:0:86173181"))
	require.Equal(t, SIDToSID3("STEAM_0:0:86173181"), SID3("[U:1:172346362]"))
	require.Equal(t, SIDToSID32("STEAM_0:0:86173181"), SID32(172346362))
	require.Equal(t, SIDToSID64("STEAM_0:0:86173181"), SID64(76561198132612090))
}

func TestSID64UnmarshalJSON(t *testing.T) {
	type tc struct {
		SteamidString string `json:"steamid_string"`
	}
	var value tc
	require.NoError(t, json.Unmarshal([]byte(`{"steamid_string":"76561197970669109"}`), &value))
	require.Equal(t, "76561197970669109", value.SteamidString)
}

func TestResolveGID(t *testing.T) {
	gid1, err := ResolveGID(context.Background(), "SQTreeHouse")
	require.NoError(t, err, "Failed to fetch gid")
	require.True(t, gid1.Valid())
	require.Equal(t, gid1, GID(103582791441572968))
	gid2, err2 := ResolveGID(context.Background(), "SQTreeHouseHJHJHSDAF")
	require.Errorf(t, err2, "Failed to fetch gid2")
	require.False(t, gid2.Valid())
}

func TestResolveSID(t *testing.T) {
	if apiKey == "" {
		t.Skip("steam_api_key unset, SetKey() required")
		return
	}
	sid1, err := ResolveSID64(context.Background(), "https://steamcommunity.com/id/SQUIRRELLY")
	require.NoError(t, err)
	require.Equal(t, sid1, SID64(76561197961279983))

	sid2, err2 := ResolveSID64(context.Background(), "https://steamcommunity.com/id/FAKEXXXXXXXXXX123123")
	require.Error(t, err2)
	require.False(t, sid2.Valid())

	sid3, err3 := ResolveSID64(context.Background(), "http://steamcommunity.com/profiles/76561197961279983")
	require.NoError(t, err3)
	require.Equal(t, sid3, SID64(76561197961279983))

	sid4, err4 := ResolveSID64(context.Background(), "[U:1:1014255]")
	require.NoError(t, err4)
	require.Equal(t, sid4, SID64(76561197961279983))

	sid5, err5 := ResolveSID64(context.Background(), "STEAM_0:1:507127")
	require.Equal(t, sid5, SID64(76561197961279983))
	require.NoError(t, err5)

	sid6, err6 := ResolveSID64(context.Background(), "")
	require.Error(t, err6)
	require.False(t, sid6.Valid())

}

func TestMain(m *testing.M) {
	key, found := os.LookupEnv("STEAM_TOKEN")
	if found {
		if e := SetKey(key); e != nil {
			fmt.Print(e.Error())
		}
	}
	os.Exit(m.Run())
}
