package extra_test

import (
	"testing"

	"github.com/leighmacdonald/steamid/v2/extra"

	"github.com/stretchr/testify/require"
)

func TestParseStatus(t *testing.T) {
	statusText := `hostname: Uncletopia | US West 2
version : 5970214/24 5970214 secure
udp/ip  : 23.239.22.163:27015  (public ip: 23.239.22.163)
steamid : [G:1:3414356] (85568392923453780)
account : not logged in  (No account specified)
map     : pl_goldrush at: 0 x, 0 y, 0 z
tags    : Uncletopia,nocrits,nodmgspread,payload
players : 11 humans, 0 bots (32 max)
edicts  : 1717 used of 2048 max
# userid name                uniqueid            connected ping loss state  adr
#   4247 "Dulahan"           [U:1:148883280]     55:09       74    0 active 1.2.64.84:27005
#   4235 "Nox"               [U:1:186134686]      1:21:18   123    0 active 1.2.212.98:27005
#   4262 "George Scrumpus"   [U:1:64274886]      17:09      118    0 active 1.2.121.68:27005
#   4254 "airbud"            [U:1:190163035]     38:49       72    0 active 1.2.246.238:27005
#   4256 "Kensei"            [U:1:119851869]     36:33       53    0 active 1.2.110.66:27005
#   4268 "Progseeks"         [U:1:191380023]     01:43      105    0 active 1.2.67.76:27005
#   4181 "Gera"              [U:1:202327912]      2:39:57   104    0 active 1.2.62.100:27005
#   4271 "A Good Idea"       [U:1:431565997]     00:41       68    0 active 1.2.104.247:27005
#   4212 "Chance The Memer"  [U:1:106864873]      1:51:58   106    0 active 1.2.215.62:27005
#   4259 "Greenwood RN"      [U:1:128375332]     24:51       67    0 active 1.2.136.246:27005
#   4246 "Frank"             [U:1:166415783]      1:01:59   133    0 active 1.2.23.197:27005
`
	ids := extra.SIDSFromStatus(statusText)
	require.NotNil(t, ids)
	require.Equal(t, len(ids), 11)

	st, err := extra.ParseStatus(statusText, true)
	require.NoError(t, err)
	require.Equal(t, "Uncletopia | US West 2", st.ServerName)
	require.Equal(t, 32, st.PlayersMax)
	require.Equal(t, 11, st.PlayersCount)
	require.Equal(t, "pl_goldrush", st.Map)
	require.Equal(t, []int{1717, 2048}, st.Edicts)
	require.Equal(t, []string{"Uncletopia", "nocrits", "nodmgspread", "payload"}, st.Tags)
	require.Equal(t, "5970214/24 5970214 secure", st.Version)
}
