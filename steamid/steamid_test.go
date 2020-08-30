package steamid

import (
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestParseStatus(t *testing.T) {
	s := `hostname: Uncletopia | US West 2
version : 5970214/24 5970214 secure
udp/ip  : 23.239.22.163:27015  (public ip: 23.239.22.163)
steamid : [G:1:3414356] (85568392923453780)
account : not logged in  (No account specified)
map     : pl_goldrush at: 0 x, 0 y, 0 z
tags    : Uncletopia,nocrits,nodmgspread,payload
players : 11 humans, 0 bots (32 max)
edicts  : 1717 used of 2048 max
# userid name                uniqueid            connected ping loss state  adr
#   4247 "Dulahan"           [U:1:148883280]     55:09       74    0 active 96.48.64.84:27005
#   4235 "Nox"               [U:1:186134686]      1:21:18   123    0 active 174.111.212.98:27005
#   4262 "George Scrumpus"   [U:1:64274886]      17:09      118    0 active 24.202.121.68:27005
#   4254 "airbud"            [U:1:190163035]     38:49       72    0 active 98.165.246.238:27005
#   4256 "Kensei"            [U:1:119851869]     36:33       53    0 active 64.201.110.66:27005
#   4268 "Progseeks"         [U:1:191380023]     01:43      105    0 active 98.109.67.76:27005
#   4181 "Gera"              [U:1:202327912]      2:39:57   104    0 active 201.105.62.100:27005
#   4271 "A Good Idea"       [U:1:431565997]     00:41       68    0 active 73.97.104.247:27005
#   4212 "Chance The Raper"  [U:1:106864873]      1:51:58   106    0 active 66.58.215.62:27005
#   4259 "Greenwood RN"      [U:1:128375332]     24:51       67    0 active 71.231.136.246:27005
#   4246 "Frank"             [U:1:166415783]      1:01:59   133    0 active 107.209.23.197:27005
`
	ids := SIDSFromStatus(s)
	require.NotNil(t, ids)
	require.Equal(t, len(ids), 11)

	st, err := ParseStatus(s, true)
	require.NoError(t, err)
	require.Equal(t, "Uncletopia | US West 2", st.ServerName)
	require.Equal(t, 32, st.PlayersMax)
	require.Equal(t, 11, st.PlayersCount)
	require.Equal(t, "pl_goldrush", st.Map)
	require.Equal(t, []int{1717, 2048}, st.Edicts)
	require.Equal(t, []string{"Uncletopia", "nocrits", "nodmgspread", "payload"}, st.Tags)
	require.Equal(t, "5970214/24 5970214 secure", st.Version)
}

func TestRandSID64(t *testing.T) {
	require.True(t, RandSID64().Valid())
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
	require.Equal(t, SID3ToSID32("[U:1:172346362]"), SID32(172346362))
	require.Equal(t, SID3ToSID64("[U:1:172346362]"), SID64(76561198132612090))
	require.Equal(t, SID3ToSID("[U:1:172346362]"), SID("STEAM_0:0:86173181"))
	require.Equal(t, SID32ToSID3(172346362), SID3("[U:1:172346362]"))
	require.Equal(t, SID32ToSID64(172346362), SID64(76561198132612090))
	require.Equal(t, SID32ToSteamID(172346362), SID("STEAM_0:0:86173181"))
	require.Equal(t, SID64ToSID3(76561198132612090), SID3("[U:1:172346362]"))
	require.Equal(t, SID64ToSID32(76561198132612090), SID32(172346362))
	require.Equal(t, SID64ToSID(76561198132612090), SID("STEAM_0:0:86173181"))
	require.Equal(t, SIDToSID3("STEAM_0:0:86173181"), SID3("[U:1:172346362]"))
	require.Equal(t, SIDToSID32("STEAM_0:0:86173181"), SID32(172346362))
	require.Equal(t, SIDToSID64("STEAM_0:0:86173181"), SID64(76561198132612090))
}

func TestPlayerSummaries(t *testing.T) {
	if apiKey == "" {
		t.Skip("steam_api_key unset, SetKey() required")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	ids := []SID64{76561198132612090, 76561197961279983, 76561197960435530}
	p, err := PlayerSummaries(ctx, ids)
	require.NoError(t, err)
	require.Equal(t, len(ids), len(p))
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

	sid2, err := ResolveSID64(context.Background(), "https://steamcommunity.com/id/FAKEXXXXXXXXXX123123")
	require.Error(t, err)
	require.False(t, sid2.Valid())

	sid3, err := ResolveSID64(context.Background(), "http://steamcommunity.com/profiles/76561197961279983")
	require.NoError(t, err)
	require.Equal(t, sid3, SID64(76561197961279983))

	sid4, err := ResolveSID64(context.Background(), "[U:1:1014255]")
	require.NoError(t, err)
	require.Equal(t, sid4, SID64(76561197961279983))

	sid5, err := ResolveSID64(context.Background(), "STEAM_0:1:507127")
	require.Equal(t, sid5, SID64(76561197961279983))
	require.NoError(t, err)

	sid6, err := ResolveSID64(context.Background(), "")
	require.Error(t, err)
	require.False(t, sid6.Valid())

}

func TestMain(m *testing.M) {
	key, found := os.LookupEnv("STEAM_TOKEN")
	if found {
		SetKey(key)
	}
	os.Exit(m.Run())
}
