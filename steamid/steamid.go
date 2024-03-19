// Package steamid provides conversion to and from all steam ID formats.
//
// If you wish to resolve vanity names like https://steamcommunity.com/id/SQUIRRELLY into
// steam id you must first obtain an API key at https://steamcommunity.com/dev/apikey.
//
// You can then set it for the package to use:
//
//		steamid.SetKey(apiKey)
//
//	With a steam api key set you can now use the following functions:
//
//		steamid.ResolveVanity()
package steamid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
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
	httpClient    *http.Client //nolint:gochecknoglobals
	reGroupIDTags = regexp.MustCompile(`<groupID64>(\w+)</groupID64>`)
	reGroupURL    = regexp.MustCompile(`steamcommunity.com/groups/(\S+)/?`)
	apiKey        string //nolint:gochecknoglobals

	// BuildVersion is replaced at compile time with the current tag or revision.
	BuildVersion = "dev"        //nolint:gochecknoglobals
	BuildCommit  = "master"     //nolint:gochecknoglobals
	BuildDate    = ""           //nolint:gochecknoglobals
	reSteam2     *regexp.Regexp //nolint:gochecknoglobals
	reSteam3     *regexp.Regexp //nolint:gochecknoglobals
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

// SteamID represents a Steam64
//
// ((Universe << 56) | (Account Type << 52) | (Instance << 32) | Account ID)
//
// This is using a string as the base type mainly to make interop with javascript/json simpler.
// There is no JSON bigint type, so when used with js the Number type gets represented as a float
// and will result in an invalid/truncated id value when decoded back to a native int64 form.
// 76561198132612090.
type SteamID struct {
	AccountID   SID32
	Instance    Instance
	AccountType AccountType
	Universe    Universe
}

func New(input any) SteamID {
	sid := SteamID{
		AccountID:   0,
		Instance:    InstanceAll,
		AccountType: AccountTypeInvalid,
		Universe:    UniverseInvalid,
	}

	var value string

	switch v := input.(type) {
	case string:
		if v == "0" {
			return sid
		}
		value = v
	case uint64:
		if v == 0 {
			return sid
		}

		value = fmt.Sprintf("%d", v)
	case int:
		if v == 0 {
			return sid
		}

		value = fmt.Sprintf("%d", v)
	case int64:
		if v == 0 {
			return sid
		}

		value = fmt.Sprintf("%d", v)
	case float64:
		if v == 0 {
			return sid
		}

		value = fmt.Sprintf("%d", int64(v))
	default:
		return sid
	}

	if value == "" {
		return sid
	}

	// steam2
	if match := reSteam2.FindStringSubmatch(value); match != nil {
		universeInt, errUniverseInt := strconv.ParseUint(match[1], 10, 64)
		if errUniverseInt != nil {
			return sid
		}

		intVal, errIntVal := strconv.ParseUint(match[2], 10, 64)
		if errIntVal != nil {
			return sid
		}

		accountIDInt, errAccountIDInt := strconv.ParseUint(match[3], 10, 64)
		if errAccountIDInt != nil {
			return sid
		}

		if universeInt > 0 {
			sid.Universe = Universe(universeInt)
		} else {
			sid.Universe = UniversePublic
		}

		sid.AccountType = AccountTypeIndividual
		sid.Instance = InstanceDesktop
		sid.AccountID = SID32((accountIDInt * 2) + intVal)

		return sid
	}

	// steam3
	if match := reSteam3.FindStringSubmatch(value); match != nil {
		ir := match[1]
		universeInt, errUniverseInt := strconv.ParseUint(match[2], 10, 64)
		if errUniverseInt != nil {
			return sid
		}

		accountIDInt, errAccountIDInt := strconv.ParseUint(match[3], 10, 64)
		if errAccountIDInt != nil {
			return sid
		}

		sid.Universe = Universe(universeInt)
		sid.AccountID = SID32(accountIDInt)
		switch ir {
		case "U":
			sid.Instance = InstanceDesktop
		case "A":
			sid.Instance = InstanceAll
		case "C":
			sid.Instance = InstanceConsole
		case "W":
			sid.Instance = InstanceWeb
		}

		switch ir {
		case "c":
			sid.Instance |= ClanMask
			sid.AccountType = AccountTypeChat
		case "L":
			sid.Instance |= Lobby
		default:
			sid.AccountType = accountTypeFromLetter(ir)
		}
		return sid
	}

	// uint64 version
	intVal, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return sid
	}

	if intVal < BaseSID {
		// 172346362 -> 76561198132612090
		sid.Universe = UniversePublic
		sid.AccountType = AccountTypeIndividual
		sid.Instance = InstanceDesktop
		sid.AccountID = SID32(intVal)
	} else {
		// 76561198132612090 -> 76561198132612090
		sid.AccountID = SID32((intVal & 0xFFFFFFFF) >> 0)
		sid.Instance = Instance(intVal >> 32 & 0xFFFFF)
		sid.AccountType = AccountType(intVal >> 52 & 0xF)
		sid.Universe = Universe(intVal >> 56)
	}

	return sid
}

func (t *SteamID) Equal(id SteamID) bool {
	return t.AccountID == id.AccountID && t.AccountType == id.AccountType && t.Instance == id.Instance && t.Universe == id.Universe
}

func (t *SteamID) String() string {
	return fmt.Sprintf("%d", t.Int64())
}

func (t *SteamID) Int64() int64 {
	return int64((uint64(t.Universe << 56)) | (uint64(t.AccountType) << 52) | (uint64(t.Instance) << 32) | uint64(t.AccountID))
}

// Valid ensures the value is at least large enough to be valid
// No further validation is done.
func (t *SteamID) Valid() bool {
	if t.AccountType <= AccountTypeInvalid || t.AccountType > AccountTypeAnonUser {
		return false
	}

	if t.Universe <= UniverseInvalid || t.Universe > UniverseDev {
		return false
	}

	if t.AccountType == AccountTypeIndividual && (t.AccountID == 0 || t.Instance > InstanceWeb) {
		return false
	}

	if t.AccountType == AccountTypeClan && (t.AccountID == 0 || t.Instance != InstanceAll) {
		return false
	}

	if t.AccountType == AccountTypeGameServer && t.AccountID == 0 {
		return false
	}

	return true
}

// Steam converts a given SID64 to a SteamID2 format.
// e.g. 76561198132612090 -> STEAM_0:0:86173181
//
// An empty SteamID (string) is returned if the process was unsuccessful.
func (t *SteamID) Steam(format bool) SID {
	if t.AccountType != AccountTypeIndividual {
		return ""
	}

	uni := t.Universe
	if !format && uni == 1 {
		uni = 0
	}

	return SID(fmt.Sprintf("STEAM_%d:%d:%d", uni, t.AccountID&1, int64(math.Floor(float64(t.AccountID)/2))))
}

// Steam3 converts a given id to a SID3 format.
// e.g. 76561198132612090 -> [U:1:172346362].
func (t *SteamID) Steam3() SID3 {
	char := t.AccountType.Letter()
	if t.Instance&ClanMask > 0 {
		char = "c"
	} else if t.Instance&Lobby > 0 {
		char = "L"
	}

	doInstance := t.AccountType == AccountTypeAnonGameServer ||
		t.AccountType == AccountTypeMultiSeat ||
		(t.AccountType == AccountTypeIndividual && t.Instance != InstanceDesktop)

	if !doInstance {
		return SID3(fmt.Sprintf("[%s:%d:%d]", char, t.Universe, t.AccountID))
	} else {
		return SID3(fmt.Sprintf("[%s:%d:%d:%d]", char, t.Universe, t.AccountID, t.Instance))
	}
}

// func (t *SteamID) IsLobby() bool {
//	return t.AccountType == AccountTypeChat && (int(t.Instance)&Lobby) || (int(t.Instance)&MMSLobby))
// }

func (t *SteamID) MarshalJSON() ([]byte, error) {
	return []byte("\"" + t.String() + "\""), nil
}

// UnmarshalJSON implements the Unmarshaler interface for steam ids. It will attempt to
// do all steam id types by calling StringToSID64.
func (t *SteamID) UnmarshalJSON(data []byte) error {
	var (
		sidInput  any
		outputSid SteamID
		err       error
	)

	if err = json.Unmarshal(data, &sidInput); err != nil {
		return errors.Join(err, ErrDecodeSID)
	}

	switch sid := sidInput.(type) {
	case string:
		outputSid = New(sid)
		if !outputSid.Valid() {
			return errors.Join(err, ErrUnmarshalStringSID)
		}

		*t = outputSid
	case int64:
		*t = New(fmt.Sprintf("%d", sid))
	default:
		return ErrInvalidSID
	}

	if !outputSid.Valid() {
		return ErrInvalidSID
	}

	return nil
}

// MarshalText implements encoding.TextMarshaler which is used by the yaml package for marshalling
func (t *SteamID) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for steam ids.
func (t *SteamID) UnmarshalYAML(node *yaml.Node) error {
	sid := New(node.Value)
	if !sid.Valid() {
		return ErrInvalidSID
	}
	*t = sid
	return nil
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

func (c Collection) Contains(sid64 SteamID) bool {
	for _, player := range c {
		if player.Int64() == sid64.Int64() {
			return true
		}
	}

	return false
}

func KeyConfigured() bool {
	return apiKey != ""
}

// SetKey will set the package global steam webapi key used for some requests
// Basic id conversion usage does not require this to be set.
//
// You can alternatively set the key with the environment variable `STEAM_TOKEN={YOUR_API_KEY`
// To get a key see: https://steamcommunity.com/dev/apikey
func SetKey(key string) error {
	if len(key) != 32 && len(key) != 0 {
		return ErrInvalidKey
	}

	apiKey = key

	return nil
}

var idGen = uint64(0) //nolint:gochecknoglobals

// RandSID64 generates a unique random (numerically) valid steamid for testing.
func RandSID64() SteamID {
	id := atomic.AddUint64(&idGen, 1)

	sid := New("")
	sid.Universe = UniversePublic
	sid.AccountType = AccountTypeIndividual
	sid.Instance = InstanceDesktop
	sid.AccountID = SID32(id)

	return sid
}

// SID64FromString will attempt to convert a Steam64 formatted string into a SID64.
func SID64FromString(steamID string) (SteamID, error) {
	if steamID == "" {
		return SteamID{}, errors.Join(ErrInvalidSID, ErrEmptyString)
	}

	i, err := strconv.ParseInt(steamID, 10, 64)
	if err != nil {
		return SteamID{}, errors.Join(err, ErrSIDConvertInt64)
	}

	sid := New(i)
	if !sid.Valid() {
		return SteamID{}, ErrInvalidSID
	}

	return sid, nil
}

// ResolveGID tries to resolve the GroupID from a group custom URL.
// NOTE This may be prone to error due to not being a real api endpoint.
func ResolveGID(ctx context.Context, groupVanityURL string) (SteamID, error) {
	m := reGroupURL.FindStringSubmatch(groupVanityURL)
	if len(m) > 0 {
		groupVanityURL = m[1]
	}

	u := "https://steamcommunity.com/groups/" + groupVanityURL + "/memberslistxml?xml=1"

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if errReq != nil {
		return SteamID{}, errors.Join(errReq, ErrRequestCreate)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return SteamID{}, errors.Join(err, ErrResponsePerform)
	}

	content, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return SteamID{}, errors.Join(errRead, ErrResponseBody)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	groupIDTags := reGroupIDTags.FindStringSubmatch(string(content))
	if len(groupIDTags) >= 2 {
		gid := New(groupIDTags[1])
		if !gid.Valid() || gid.AccountType != AccountTypeClan {
			return SteamID{}, ErrInvalidGID
		}

		return gid, nil
	}

	return SteamID{}, ErrResolveVanityGID
}

type vanityURLResponse struct {
	Response struct {
		SteamID SteamID `json:"steamid"`
		Success int     `json:"success"`
	} `json:"response"`
}

// ResolveVanity attempts to resolve the underlying SID64 of a users vanity url name
// This only accepts the name or last portion of the /id/ profile link
// For https://steamcommunity.com/id/SQUIRRELLY the value is SQUIRRELLY.
func ResolveVanity(ctx context.Context, query string) (SteamID, error) {
	if apiKey == "" {
		return SteamID{}, ErrNoAPIKey
	}

	u := urlVanity + url.Values{"key": {apiKey}, "vanityurl": {query}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return SteamID{}, errors.Join(err, ErrRequestCreate)
	}

	resp, errDo := httpClient.Do(req)
	if errDo != nil {
		return SteamID{}, errors.Join(errDo, ErrResponsePerform)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var vanityResp vanityURLResponse
	if errUnmarshal := json.NewDecoder(resp.Body).Decode(&vanityResp); err != nil {
		return SteamID{}, errors.Join(errUnmarshal, ErrDecodeSID)
	}

	if vanityResp.Response.Success != 1 {
		return SteamID{}, fmt.Errorf("%w: %d", ErrInvalidStatusCode, vanityResp.Response.Success)
	}

	if !vanityResp.Response.SteamID.Valid() {
		return SteamID{}, fmt.Errorf("%w: %s", ErrInvalidSID, vanityResp.Response.SteamID.String())
	}

	return vanityResp.Response.SteamID, nil
}

// ResolveSID64 tries to retrieve a SteamID64 using a query (search).
//
// If an error occurs or the SteamID was unable to be resolved from the query
// then am error is returned.
// TODO try and resolve len(17) && len(9) failed conversions as vanity.
func ResolveSID64(ctx context.Context, query string) (SteamID, error) {
	query = strings.ReplaceAll(query, " ", "")
	if strings.Contains(query, "steamcommunity.com/profiles/") {
		if string(query[len(query)-1]) == "/" {
			query = query[0 : len(query)-1]
		}

		output, err := strconv.ParseInt(query[strings.Index(query, "steamcommunity.com/profiles/")+len("steamcommunity.com/profiles/"):], 10, 64)
		if err != nil {
			return SteamID{}, errors.Join(err, ErrInvalidQueryValue)
		}

		// query = strings.Replace(query, "/", "", -1)
		if len(strconv.FormatInt(output, 10)) != 17 {
			return SteamID{}, errors.Join(err, ErrInvalidQueryLen)
		}

		return New(output), nil
	} else if strings.Contains(query, "steamcommunity.com/id/") {
		if string(query[len(query)-1]) == "/" {
			query = query[0 : len(query)-1]
		}
		query = query[strings.Index(query, "steamcommunity.com/id/")+len("steamcommunity.com/id/"):]
		return ResolveVanity(ctx, query)
	}

	s := New(query)
	if s.Valid() {
		return s, nil
	}

	return ResolveVanity(ctx, query)
}

// ParseString attempts to parse any strings of any known format within the body to a common SID64 format.
func ParseString(body string) []SteamID {
	freSID := regexp.MustCompile(`STEAM_0:[01]:[0-9][0-9]{0,8}`)
	freSID64 := regexp.MustCompile(`7656119\d{10}`)
	freSID3 := regexp.MustCompile(`\[U:1:\d+]`)

	// Store only unique entries
	found := make(map[int64]bool)
	m0 := freSID.FindAllStringSubmatch(body, -1)
	m1 := freSID64.FindAllStringSubmatch(body, -1)
	m2 := freSID3.FindAllStringSubmatch(body, -1)

	for _, i := range m0 {
		sid := New(i[0])
		found[sid.Int64()] = true
	}

	for _, i := range m1 {
		s := New(i[0])
		if !s.Valid() {
			continue
		}

		found[s.Int64()] = true
	}

	for _, i := range m2 {
		sid := New(i[0])
		found[sid.Int64()] = true
	}

	var ids []SteamID
	for k := range found {
		ids = append(ids, New(k))
	}

	var uniq []SteamID
	for _, id := range ids {
		isFound := false
		for _, uid := range uniq {
			if uid.Int64() == id.Int64() {
				isFound = true
				break
			}
		}

		if isFound {
			continue
		}

		uniq = append(uniq, id)
	}
	return ids
}

func init() {
	reSteam2 = regexp.MustCompile(`^STEAM_([0-5]):([0-1]):([0-9]+)$`)
	reSteam3 = regexp.MustCompile(`^\[([a-zA-Z]):([0-5]):([0-9]+)(:[0-9]+)?]$`)
	if t, found := os.LookupEnv("STEAM_TOKEN"); found && t != "" {
		if err := SetKey(t); err != nil {
			panic(err)
		}
	}

	httpClient = &http.Client{
		Timeout: time.Second * 10,
	}
}
