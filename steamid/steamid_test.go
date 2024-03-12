package steamid_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestRandSID64(t *testing.T) {
	t.Parallel()

	sid := steamid.RandSID64()
	require.True(t, sid.Valid())
}

func TestSID64FromString(t *testing.T) {
	t.Parallel()

	v, err := steamid.SID64FromString("76561198132612090")
	require.NoError(t, err)

	sid := steamid.New(76561198132612090)
	require.Equal(t, sid, v)

	v2, err2 := steamid.SID64FromString("asdf")
	require.Error(t, err2)
	require.Equal(t, steamid.New(""), v2)

	v3, err3 := steamid.SID64FromString("")
	require.Error(t, err3)
	require.Equal(t, steamid.New("0"), v3)
}

func TestGIDFromString(t *testing.T) {
	t.Parallel()

	g0 := steamid.New("103582791441572968")
	require.True(t, g0.Valid() && g0.AccountType == steamid.AccountTypeClan)
	require.Equal(t, steamid.New(103582791441572968), g0)

	g1 := steamid.New("asdf")
	require.False(t, g1.Valid())
	require.Equal(t, steamid.New(""), g1)

	g2 := steamid.New("")
	require.False(t, g2.Valid())
}

func TestParseString(t *testing.T) {
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

	ids := steamid.ParseString(testBody)
	require.Len(t, ids, 8) // 2 duplicated
}

func TestConversions(t *testing.T) {
	t.Parallel()

	sid := steamid.New(76561199127271263)
	require.Equal(t, steamid.SID3("[U:1:1167005535]"), sid.Steam3())
	require.Equal(t, steamid.SID("STEAM_0:1:583502767"), sid.Steam(false))
	require.Equal(t, steamid.SID32(1167005535), sid.AccountID)

	i := steamid.New(76561199127271263)
	require.Equal(t, i.Steam3(), steamid.SID3("[U:1:1167005535]"))

	ii := steamid.New("[U:1:172346362]")
	require.Equal(t, ii.AccountID, steamid.SID32(172346362))
	require.Equal(t, steamid.New("[U:1:172346362]"), steamid.New(76561198132612090))

	a := steamid.New("[U:1:172346362]")
	require.True(t, a.Equal(steamid.New("STEAM_0:0:86173181")))
	require.Equal(t, ii.Steam(false), steamid.SID("STEAM_0:0:86173181"))
}

func TestJSON(t *testing.T) {
	t.Parallel()

	type testFormats struct {
		Quoted steamid.SteamID `json:"quoted"`
	}

	var out testFormats
	require.NoError(t, json.Unmarshal([]byte(`{"quoted":"76561197970669109"}`), &out))

	expected := steamid.New(76561197970669109)
	require.Equal(t, expected, out.Quoted, "Quoted value invalid")

	body, errMarshal := json.Marshal(&expected)
	require.NoError(t, errMarshal)
	require.Equal(t, []byte("\"76561197970669109\""), body)

	type testGIDResp struct {
		GID steamid.SteamID `json:"gid"`
	}

	var r testGIDResp
	require.NoError(t, json.Unmarshal([]byte(`{"gid":"103582791441572968"}`), &r))
	expectedGID := steamid.New(103582791441572968)

	require.Equal(t, expectedGID.Int64(), r.GID.Int64())
}

func TestResolveGID(t *testing.T) {
	t.Parallel()

	gid1, err := steamid.ResolveGID(context.Background(), "SQ_Stream")

	require.NoError(t, err, "Failed to fetch gid")
	require.True(t, gid1.Valid())
	require.Equal(t, gid1, steamid.New(103582791441572968))

	gid2, err2 := steamid.ResolveGID(context.Background(), "SQTreeHouseHJHJHSDAF")

	require.Errorf(t, err2, "Failed to fetch gid2")
	require.False(t, gid2.Valid())
}

func TestResolveSID(t *testing.T) {
	t.Parallel()

	if !steamid.KeyConfigured() {
		t.Skip("steam_api_key unset, SetKey() required")

		return
	}

	sid1, err := steamid.ResolveSID64(context.Background(), "https://steamcommunity.com/id/SQUIRRELLY")
	require.NoError(t, err)
	require.Equal(t, sid1, steamid.New(76561197961279983))

	sid2, err2 := steamid.ResolveSID64(context.Background(), "https://steamcommunity.com/id/FAKEXXXXXXXXXX123123")
	require.Error(t, err2)
	require.False(t, sid2.Valid())

	sid3, err3 := steamid.ResolveSID64(context.Background(), "http://steamcommunity.com/profiles/76561197961279983")
	require.NoError(t, err3)
	require.Equal(t, sid3, steamid.New(76561197961279983))

	sid4, err4 := steamid.ResolveSID64(context.Background(), "[U:1:1014255]")
	require.NoError(t, err4)
	require.Equal(t, sid4, steamid.New(76561197961279983))

	sid5, err5 := steamid.ResolveSID64(context.Background(), "STEAM_0:1:507127")
	require.Equal(t, sid5, steamid.New(76561197961279983))
	require.NoError(t, err5)

	sid6, err6 := steamid.ResolveSID64(context.Background(), "")
	require.Error(t, err6)
	require.False(t, sid6.Valid())
}

func TestMain(m *testing.M) {
	key, found := os.LookupEnv("STEAM_TOKEN")

	if found {
		if e := steamid.SetKey(key); e != nil {
			panic(e.Error())
		}
	}

	os.Exit(m.Run())
}
