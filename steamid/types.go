package steamid

import (
	"errors"
	"slices"
)

const (
	urlVanity    = "https://api.steampowered.com/ISteamUser/ResolveVanityURL/v0001/?"
	BaseGID      = uint64(103582791429521408)
	BaseSID      = uint64(76561197960265728)
	InstanceMask = 0x000FFFFF
	ClanMask     = (InstanceMask + 1) >> 1
	Lobby        = (InstanceMask + 1) >> 2
	MMSLobby     = (InstanceMask + 1) >> 3
)

var (
	// ErrNoAPIKey is returned for functions that require an API key to use when one has not been set.
	ErrNoAPIKey = errors.New("no steam web api key, to obtain one see: " +
		"https://steamcommunity.com/dev/apikey and call steamid.SetKey()")
	ErrInvalidKey         = errors.New("invalid steam api key length, must be 32 chars or 0 to remove it")
	ErrInvalidSID         = errors.New("invalid steam id")
	ErrEmptyString        = errors.New("invalid id, string empty")
	ErrSIDConvertInt64    = errors.New("failed to convert id to int64")
	ErrInvalidGID         = errors.New("invalid gid")
	ErrDecodeSID          = errors.New("could not decode steamid value")
	ErrUnmarshalStringSID = errors.New("failed to unmarshal string to SteamID")
	ErrRequestCreate      = errors.New("failed to create request")
	ErrInvalidStatusCode  = errors.New("invalid status code")
	ErrResponsePerform    = errors.New("failed to perform request")
	ErrResponseBody       = errors.New("failed to read response body")
	ErrResolveVanityGID   = errors.New("failed to resolve group vanity name")
	ErrInvalidQueryValue  = errors.New("invalid query value")
	ErrInvalidQueryLen    = errors.New("invalid value length")
)

// AppID is the id associated with games/apps.
type AppID uint32

// SID represents a SteamID
// STEAM_0:0:86173181.
type SID string

// Universe describes the 6 known steam universe
// Universes 0 to 3 are common, 4 Dev not exist in all games, 5 RC is removed out from some source files "// no such universe anymore".
type Universe int

const (
	UniverseInvalid Universe = iota
	UniversePublic
	UniverseBeta
	UniverseInternal
	UniverseDev
	UniverseRC
)

func (u Universe) String() string {
	switch u {
	case UniversePublic:
		return "Public"
	case UniverseBeta:
		return "Beta"
	case UniverseInternal:
		return "Internal"
	case UniverseDev:
		return "Dev"
	case UniverseRC:
		return "RC"
	case UniverseInvalid:
		fallthrough
	default:
		return "Individual/Unspecified"
	}
}

// AccountType is split into 10 types for a Steam account, of which 4 can be created today.
// Users of an "Individual" account are temporarily referred to as having a "Pending" account, which has a
// textual representation of "STEAM_ID_PENDING" until their account credentials are verified with Steam's
// authentication servers, a process usually complete by the time a server is fully connected to. Accounts of the
// type "Invalid" have a textual representation of "UNKNOWN" and are used for bots and accounts which do not belong
// to another class.
//
// Multi-user chats use the "T" character. Steam group (clan) chats use the "c" character. Steam lobbies
// use Chat IDs and use the "L" character.
type AccountType int

const (
	AccountTypeInvalid AccountType = iota
	AccountTypeIndividual
	AccountTypeMultiSeat
	AccountTypeGameServer
	AccountTypeAnonGameServer
	AccountTypePending
	AccountTypeContentServer
	AccountTypeClan
	AccountTypeChat
	AccountTypeP2PSuperSeeder
	AccountTypeAnonUser
)

func (ac AccountType) String() string {
	switch ac {
	case AccountTypeIndividual:
		return "Individual"
	case AccountTypeMultiSeat:
		return "MultiSeat"
	case AccountTypeGameServer:
		return "Game Server"
	case AccountTypeAnonGameServer:
		return "Anon Game Server"
	case AccountTypePending:
		return "Pending"
	case AccountTypeContentServer:
		return "Content Server"
	case AccountTypeClan:
		return "Clan"
	case AccountTypeChat:
		return "Chat"
	case AccountTypeP2PSuperSeeder:
		return "P2P SuperSeeder"
	case AccountTypeAnonUser:
		return "Anon User"
	case AccountTypeInvalid:
		fallthrough
	default:
		return "Invalid"
	}
}

func (ac AccountType) Letter() string {
	switch ac {
	case AccountTypeIndividual:
		return "U"
	case AccountTypeMultiSeat:
		return "M"
	case AccountTypeGameServer:
		return "G"
	case AccountTypeAnonGameServer:
		return "A"
	case AccountTypePending:
		return "P"
	case AccountTypeContentServer:
		return "C"
	case AccountTypeClan:
		return "g"
	case AccountTypeChat:
		return "T"
	case AccountTypeP2PSuperSeeder:
		return ""
	case AccountTypeAnonUser:
		return "a"
	case AccountTypeInvalid:
		fallthrough
	default:
		return "I"
	}
}

func accountTypeFromLetter(l string) AccountType {
	switch l {
	case "U":
		return AccountTypeIndividual
	case "M":
		return AccountTypeMultiSeat
	case "G":
		return AccountTypeGameServer
	case "A":
		return AccountTypeAnonGameServer
	case "P":
		return AccountTypePending
	case "C":
		return AccountTypeContentServer
	case "g":
		return AccountTypeClan
	case "T":
		return AccountTypeChat
	case "":
		return AccountTypeP2PSuperSeeder
	case "a":
		return AccountTypeAnonUser
	case "I":
		return AccountTypeInvalid
	case "i":
		fallthrough
	default:
		return AccountTypeInvalid
	}
}

type Instance int

const (
	InstanceAll Instance = iota
	InstanceDesktop
	InstanceConsole
	InstanceWeb
)

func (i Instance) String() string {
	switch i {
	case InstanceAll:
		return "All"
	case InstanceDesktop:
		return "Desktop"
	case InstanceConsole:
		return "Console"
	case InstanceWeb:
		return "Web"
	default:
		return ""
	}
}

// SID32 represents a Steam32
// 172346362.
type SID32 uint32

// SID3 represents a Steam3
// [U:1:172346362].
type SID3 string

type Collection []SteamID

func (c Collection) ToStringSlice() []string {
	var s []string

	for _, st := range c {
		s = append(s, st.String())
	}

	return s
}

func (c Collection) ToInt64Slice() []int64 {
	var s []int64

	for _, st := range c {
		s = append(s, st.Int64())
	}

	return s
}

func (c Collection) Contains(sid64 SteamID) bool {
	return slices.ContainsFunc(c, func(id SteamID) bool {
		return id.Int64() == sid64.Int64()
	})
}
