package extra

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	reStatusID         = regexp.MustCompile(`"(.+?)"\s+(\[U:\d+:\d+]|STEAM_\d:\d:\d+)`)
	reStatusPlayerFull = regexp.MustCompile(`^#\s+(\d+)\s+"(.+?)"\s+(\[U:\d:\d+])\s+(.+?)\s+(\d+)\s+(\d+)\s+(.+?)\s(.+?):(.+?)$`)
	reStatusPlayer     = regexp.MustCompile(`^#\s+(\d+)\s+"(.+?)"\s+(\[U:\d:\d+])\s+(\d+:\d+)\s+(\d+)\s+(\d+)\s+(.+?)$`)
)

var (
	ErrParsePlayers    = errors.New("failed to parse players")
	ErrParseEdict      = errors.New("failed to parse edicts")
	ErrParseEdictTotal = errors.New("failed to parse total edicts")
	ErrParseUserID     = errors.New("failed to parse user id")
	ErrParsePing       = errors.New("failed to parse ping")
	ErrParseLoss       = errors.New("failed to parse loss")
	ErrParseSeconds    = errors.New("failed to parse seconds")
	ErrParseDuration   = errors.New("failed to parse duration")
	ErrParseIP         = errors.New("failed to parse ip")
	ErrParsePort       = errors.New("failed to parse port")
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
	SID           steamid.SteamID
	ConnectedTime time.Duration
	Ping          int
	Loss          int
	State         string
	IP            net.IP
	Port          int
}

// SIDSFromStatus will parse the output of the console command `status` and return a
// set of SID64s representing all the players.
func SIDSFromStatus(text string) []steamid.SteamID {
	var ids []steamid.SteamID

	found := reStatusID.FindAllString(text, -1)

	if found == nil {
		return nil
	}

	for _, strID := range found {
		ids = append(ids, steamid.New(strID))
	}

	return ids
}

func parseMaxPlayers(part string) int {
	ps := strings.Split(strings.ReplaceAll(part, "(", ""), " ")

	m, errPlayers := strconv.ParseUint(ps[4], 10, 64)
	if errPlayers != nil {
		return -1
	}

	return int(m)
}

func parseEdits(part string) []int {
	ed := strings.Split(part, " ")

	l, errEdictCount := strconv.ParseUint(ed[0], 10, 64)
	if errEdictCount != nil {
		return []int{-1, -1}
	}

	m, errEdictTotal := strconv.ParseUint(ed[3], 10, 64)
	if errEdictTotal != nil {
		return []int{-1, -1}
	}

	return []int{int(l), int(m)}
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
				if maxPlayers := parseMaxPlayers(parts[1]); maxPlayers > 0 {
					s.PlayersMax = maxPlayers
				}
			case "edicts":
				if ed := parseEdits(parts[1]); ed[0] > 0 && ed[1] > 0 {
					s.Edicts = ed
				}
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
					return Status{}, errors.Join(errUserID, ErrParseUserID)
				}

				ping, err2 := strconv.ParseUint(m[5], 10, 64)
				if err2 != nil {
					return Status{}, errors.Join(err2, ErrParsePing)
				}

				loss, err3 := strconv.ParseUint(m[6], 10, 64)
				if err3 != nil {
					return Status{}, errors.Join(err3, ErrParseLoss)
				}

				tp := strings.Split(m[4], ":")

				for i, j := 0, len(tp)-1; i < j; i, j = i+1, j-1 {
					tp[i], tp[j] = tp[j], tp[i]
				}

				var totalSec int

				for i, vStr := range tp {
					v, errUint := strconv.ParseUint(vStr, 10, 64)
					if errUint != nil {
						return Status{}, errors.Join(errUint, ErrParseSeconds)
					}

					totalSec += int(v) * []int{1, 60, 3600}[i]
				}

				dur, errDur := time.ParseDuration(fmt.Sprintf("%ds", totalSec))

				if errDur != nil {
					return Status{}, errors.Join(errDur, ErrParseDuration)
				}

				p := Player{
					UserID:        int(userID),
					Name:          m[2],
					SID:           steamid.New(m[3]),
					ConnectedTime: dur,
					Ping:          int(ping),
					Loss:          int(loss),
					State:         m[7],
				}

				if full {
					port, errFull := strconv.ParseUint(m[9], 10, 64)
					if errFull != nil {
						return Status{}, errors.Join(errFull, ErrParsePort)
					}

					ip := net.ParseIP(m[8])
					if ip == nil {
						return Status{}, ErrParseIP
					}

					p.IP = ip
					p.Port = int(port)
				}

				s.Players = append(s.Players, p)
			}
		}
	}

	s.PlayersCount = len(s.Players)

	return s, nil
}
