package extra

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

var (
	reStatusID         = regexp.MustCompile(`"(.+?)"\s+(\[U:\d+:\d+]|STEAM_\d:\d:\d+)`)
	reStatusPlayerFull = regexp.MustCompile(`^#\s+(\d+)\s+"(.+?)"\s+(\[U:\d:\d+])\s+(.+?)\s+(\d+)\s+(\d+)\s+(.+?)\s(.+?):(.+?)$`)
	reStatusPlayer     = regexp.MustCompile(`^#\s+(\d+)\s+"(.+?)"\s+(\[U:\d:\d+])\s+(\d+:\d+)\s+(\d+)\s+(\d+)\s+(.+?)$`)
)

// Status represents the data from the `status` rcon/console command.
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

// Player represents all the available data for a player in a `status` output table.
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
// set of SID64s representing all the players.
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
// This only works for status commands via RCON/CLI.
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

				m, errPlayers := strconv.ParseUint(ps[4], 10, 64)
				if errPlayers != nil {
					return Status{}, errors.Wrap(errPlayers, "Failed to parse players")
				}

				s.PlayersMax = int(m)
			case "edicts":
				ed := strings.Split(parts[1], " ")

				l, errEdictCount := strconv.ParseUint(ed[0], 10, 64)
				if errEdictCount != nil {
					return Status{}, errors.Wrap(errEdictCount, "Failed to parse edict count")
				}

				m, errEdictTotal := strconv.ParseUint(ed[3], 10, 64)
				if errEdictTotal != nil {
					return Status{}, errors.Wrap(errEdictTotal, "Failed to parse edict total")
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
				userID, errUserID := strconv.ParseUint(m[1], 10, 64)
				if errUserID != nil {
					return Status{}, errors.Wrap(errUserID, "Failed to parse userid")
				}

				ping, err2 := strconv.ParseUint(m[5], 10, 64)
				if err2 != nil {
					return Status{}, errors.Wrap(err2, "Failed to parse ping")
				}

				loss, err3 := strconv.ParseUint(m[6], 10, 64)
				if err3 != nil {
					return Status{}, errors.Wrap(err3, "Failed to parse loss")
				}

				tp := strings.Split(m[4], ":")
				for i, j := 0, len(tp)-1; i < j; i, j = i+1, j-1 {
					tp[i], tp[j] = tp[j], tp[i]
				}

				var totalSec int
				for i, vStr := range tp {
					v, errUint := strconv.ParseUint(vStr, 10, 64)
					if errUint != nil {
						return Status{}, errors.Wrap(errUint, "Failed to parse total seconds")
					}
					totalSec += int(v) * []int{1, 60, 3600}[i]
				}

				dur, errDur := time.ParseDuration(fmt.Sprintf("%ds", totalSec))
				if errDur != nil {
					return Status{}, errors.Wrap(errDur, "Failed to parse duration")
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
					port, errFull := strconv.ParseUint(m[9], 10, 64)
					if errFull != nil {
						return Status{}, errors.Wrap(errFull, "Failed to parse uint")
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
