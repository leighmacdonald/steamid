package extra_test

import (
	"testing"

	"github.com/leighmacdonald/steamid/v4/extra"
	"github.com/stretchr/testify/require"
)

func TestParseStatus(t *testing.T) {
	t.Parallel()

	statusText := `hostname: Uncletopia | US West 2
version : 5970214/24 5970214 secure
udp/ip  : 23.239.22.163:27015  (public ip: 23.239.22.163)
steamid : [G:1:3414356] (85568392923453780)
account : not logged in  (No account specified)
map     : pl_goldrush at: 0 x, 0 y, 0 z
tags    : Uncletopia,nocrits,nodmgspread,payload
sourcetv:  108.181.62.21:27015, delay 0.0s  (local: 108.181.62.21:27016)
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

	parsedStatus, err := extra.ParseStatus(statusText, true)
	require.NoError(t, err)

	require.Equal(t, "23.239.22.163", parsedStatus.IPInfo.PublicIP)
	require.Equal(t, 27015, parsedStatus.IPInfo.PublicPort)

	require.Equal(t, "Uncletopia | US West 2", parsedStatus.ServerName)

	require.Equal(t, "108.181.62.21", parsedStatus.IPInfo.SourceTVIP)
	require.Equal(t, 27015, parsedStatus.IPInfo.SourceTVFPort)
	require.Equal(t, "108.181.62.21", parsedStatus.IPInfo.SourceTVLocalIP)
	require.Equal(t, 27016, parsedStatus.IPInfo.SourceTVLocalPort)
	require.False(t, parsedStatus.IPInfo.SDR)

	require.Equal(t, 32, parsedStatus.PlayersMax)
	require.Equal(t, 11, parsedStatus.PlayersCount)
	require.Equal(t, "pl_goldrush", parsedStatus.Map)
	require.Equal(t, []int{1717, 2048}, parsedStatus.Edicts)
	require.Equal(t, []string{"Uncletopia", "nocrits", "nodmgspread", "payload"}, parsedStatus.Tags)
	require.Equal(t, "5970214/24 5970214 secure", parsedStatus.Version)
}

func TestParseStatusSDR(t *testing.T) {
	t.Parallel()

	statusText := `hostname: Uncletopia | US West 2
version : 5970214/24 5970214 secure
udp/ip  : 169.254.176.141:13176  (local: 192.168.0.201:27015)  (public IP from Steam: 1.233.33.1)
steamid : [G:1:3414356] (85568392923453780)
account : not logged in  (No account specified)
map     : pl_goldrush at: 0 x, 0 y, 0 z
tags    : Uncletopia,nocrits,nodmgspread,payload
sourcetv:  169.254.176.141:13176, delay 0.0s  (local: 192.168.0.201:27016)
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

	parsedStatus, err := extra.ParseStatus(statusText, true)
	require.NoError(t, err)
	require.Equal(t, "Uncletopia | US West 2", parsedStatus.ServerName)
	require.Equal(t, 32, parsedStatus.PlayersMax)
	require.Equal(t, 11, parsedStatus.PlayersCount)
	require.Equal(t, "pl_goldrush", parsedStatus.Map)
	require.Equal(t, []int{1717, 2048}, parsedStatus.Edicts)
	require.Equal(t, []string{"Uncletopia", "nocrits", "nodmgspread", "payload"}, parsedStatus.Tags)
	require.Equal(t, "5970214/24 5970214 secure", parsedStatus.Version)

	require.Equal(t, "169.254.176.141", parsedStatus.IPInfo.SourceTVIP)
	require.Equal(t, 13176, parsedStatus.IPInfo.SourceTVFPort)
	require.Equal(t, "192.168.0.201", parsedStatus.IPInfo.SourceTVLocalIP)
	require.Equal(t, 27016, parsedStatus.IPInfo.SourceTVLocalPort)
	require.True(t, parsedStatus.IPInfo.SDR)
}
