package extra

import (
	"fmt"
	"github.com/leighmacdonald/steamid/steamid"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reStatusID         *regexp.Regexp
	reStatusPlayerFull *regexp.Regexp
	reStatusPlayer     *regexp.Regexp
)

func init() {
	reStatusID = regexp.MustCompile(`"(.+?)"\s+(\[U:\d+:\d+]|STEAM_\d:\d:\d+)`)
	reStatusPlayer = regexp.MustCompile(`^#\s+(\d+)\s+"(.+?)"\s+(\[U:\d:\d+])\s+(\d+:\d+)\s+(\d+)\s+(\d+)\s+(.+?)$`)
	reStatusPlayerFull = regexp.MustCompile(`^#\s+(\d+)\s+"(.+?)"\s+(\[U:\d:\d+])\s+(.+?)\s+(\d+)\s+(\d+)\s+(.+?)\s(.+?):(.+?)$`)
}

type Status struct {
	PlayersCount int
	PlayersMax   int
	ServerName   string
	Version      string
	Edicts       []int
	Tags         []string
	Map          string
	Players      []Player
}

type Player struct {
	UserID        int
	Name          string
	SID           steamid.SID64
	ConnectedTime time.Duration
	Ping          int
	Loss          int
	State         string
	IP            net.IP
	Port          int
}

// SIDSFromStatus will parse the output of the console command `status` and return a
// set of SID64s representing all the players
func SIDSFromStatus(text string) []steamid.SID64 {
	var ids []steamid.SID64
	found := reStatusID.FindAllString(text, -1)
	if found == nil {
		return nil
	}
	for _, strID := range found {
		ids = append(ids, steamid.SID3ToSID64(steamid.SID3(strID)))
	}
	return ids
}

// ParseStatus will parse a status command output into a struct
// If full is true, it will also parse the address/port of the player.
// This only works for status commands via RCON/CLI
func ParseStatus(status string, full bool) (Status, error) {
	var s Status
	for _, line := range strings.Split(status, "\n") {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			switch strings.TrimRight(parts[0], " ") {
			case "hostname":
				s.ServerName = parts[1]
			case "version":
				s.Version = parts[1]
			case "map":
				s.Map = strings.Split(parts[1], " ")[0]
			case "tags":
				s.Tags = strings.Split(parts[1], ",")
			case "players":
				ps := strings.Split(strings.ReplaceAll(parts[1], "(", ""), " ")
				m, err := strconv.ParseUint(ps[4], 10, 64)
				if err != nil {
					return Status{}, err
				}
				s.PlayersMax = int(m)
			case "edicts":
				ed := strings.Split(parts[1], " ")
				l, err := strconv.ParseUint(ed[0], 10, 64)
				if err != nil {
					return Status{}, err
				}
				m, err := strconv.ParseUint(ed[3], 10, 64)
				if err != nil {
					return Status{}, err
				}
				s.Edicts = []int{int(l), int(m)}
			}
			continue
		} else {
			var m []string
			if full {
				m = reStatusPlayerFull.FindStringSubmatch(line)
			} else {
				m = reStatusPlayer.FindStringSubmatch(line)
			}
			if (!full && len(m) == 8) || (full && len(m) == 10) {
				userID, err := strconv.ParseUint(m[1], 10, 64)
				if err != nil {
					return Status{}, err
				}
				ping, err := strconv.ParseUint(m[5], 10, 64)
				if err != nil {
					return Status{}, err
				}
				loss, err := strconv.ParseUint(m[6], 10, 64)
				if err != nil {
					return Status{}, err
				}
				tp := strings.Split(m[4], ":")
				for i, j := 0, len(tp)-1; i < j; i, j = i+1, j-1 {
					tp[i], tp[j] = tp[j], tp[i]
				}
				var totalSec int
				for i, vStr := range tp {
					v, err := strconv.ParseUint(vStr, 10, 64)
					if err != nil {
						return Status{}, err
					}
					totalSec += int(v) * []int{1, 60, 3600}[i]
				}
				dur, err := time.ParseDuration(fmt.Sprintf("%ds", totalSec))
				if err != nil {
					return Status{}, err
				}
				p := Player{
					UserID:        int(userID),
					Name:          m[2],
					SID:           steamid.SID3ToSID64(steamid.SID3(m[3])),
					ConnectedTime: dur,
					Ping:          int(ping),
					Loss:          int(loss),
					State:         m[7],
				}
				if full {
					port, err := strconv.ParseUint(m[9], 10, 64)
					if err != nil {
						return Status{}, err
					}
					p.IP = net.ParseIP(m[8])
					p.Port = int(port)
				}
				s.Players = append(s.Players, p)
			}
		}
	}
	s.PlayersCount = len(s.Players)
	return s, nil
}
