// Package steamid provides conversion to and from all steam ID formats.
//
// Resolving vanity names like https://steamcommunity.com/id/SQUIRRELLY
// is done by using https://steamcommunity/id/SQUIRRELLY/?xml=1 URL
// If you want to avoid potentially being rate limited you can set a Steam Web API key
// which can be obtained from https://steamcommunity.com/dev/apikey
//
// You can then set it for the package to use:
//
//		steamid.SetKey(apiKey)
//
//	With a steam api key set you can now use the following functions:
//
//		steamid.Resolve()
//		steamid.ResolveVanity()
package steamid

import (
	"context"
	"database/sql/driver"
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

var (
	httpClient    *http.Client //nolint:gochecknoglobals
	reGroupIDTags = regexp.MustCompile(`<groupID64>(\w+)</groupID64>`)
	reGroupURL    = regexp.MustCompile(`steamcommunity.com/groups/(\S+)/?`)
	reProfileID   = regexp.MustCompile(`<steamID64>(\w+)</steamID64>`)
	apiKey        string //nolint:gochecknoglobals

	// BuildVersion is replaced at compile time with the current tag or revision.
	BuildVersion = "dev"        //nolint:gochecknoglobals
	BuildCommit  = "master"     //nolint:gochecknoglobals
	BuildDate    = ""           //nolint:gochecknoglobals
	reSteam2     *regexp.Regexp //nolint:gochecknoglobals
	reSteam3     *regexp.Regexp //nolint:gochecknoglobals
)

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

func fromSteam2Strings(match []string) SteamID {
	sid := SteamID{
		AccountID:   0,
		Instance:    InstanceAll,
		AccountType: AccountTypeInvalid,
		Universe:    UniverseInvalid,
	}
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
		sid.Universe = Universe(universeInt) //nolint:gosec
	} else {
		sid.Universe = UniversePublic
	}

	sid.AccountType = AccountTypeIndividual
	sid.Instance = InstanceDesktop
	sid.AccountID = SID32((accountIDInt * 2) + intVal) //nolint:gosec

	return sid
}

func fromSteam3Strings(match []string) SteamID {
	sid := SteamID{AccountID: 0, Instance: InstanceAll, AccountType: AccountTypeInvalid, Universe: UniverseInvalid}
	ir := match[1]
	universeInt, errUniverseInt := strconv.ParseUint(match[2], 10, 64)
	if errUniverseInt != nil {
		return sid
	}

	accountIDInt, errAccountIDInt := strconv.ParseUint(match[3], 10, 64)
	if errAccountIDInt != nil {
		return sid
	}

	sid.Universe = Universe(universeInt) //nolint:gosec
	sid.AccountID = SID32(accountIDInt)  //nolint:gosec
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

func fromUInt64(intVal uint64) SteamID {
	// 76561198132612090 -> 76561198132612090
	sid := SteamID{AccountID: 0, Instance: InstanceAll, AccountType: AccountTypeInvalid, Universe: UniverseInvalid}
	sid.Universe = UniversePublic
	sid.AccountType = AccountTypeIndividual
	sid.Instance = InstanceDesktop
	sid.AccountID = SID32(intVal) //nolint:gosec

	return sid
}

func fromAccountID(accountID uint64) SteamID {
	// 172346362 -> 76561198132612090
	sid := SteamID{AccountID: 0, Instance: InstanceAll, AccountType: AccountTypeInvalid, Universe: UniverseInvalid}
	sid.AccountID = SID32((accountID & 0xFFFFFFFF) >> 0) //nolint:gosec
	sid.Instance = Instance(accountID >> 32 & 0xFFFFF)   //nolint:gosec
	sid.AccountType = AccountType(accountID >> 52 & 0xF) //nolint:gosec
	sid.Universe = Universe(accountID >> 56)             //nolint:gosec

	return sid
}

var invalidSID = SteamID{AccountID: 0, Instance: InstanceAll, AccountType: AccountTypeInvalid, Universe: UniverseInvalid} //nolint:gochecknoglobals

// New accepts the following forms of steamid:
//
// Steam64:
// - "76561198045011302"
// - int64(76561198045011302)
// - uint64(76561198045011302)
// Steam3:
// - "[U:1:84745574]"
// Steam:
// - "STEAM_0:0:42372787"
// AccountID:
// - int(84745574)
// - int32(84745574)
// - int64(84745574)
//
// Returned SteamID should be verified with the SteamID.Valid method.
func New(input any) SteamID {
	var value string

	switch v := input.(type) {
	case string:
		if v == "0" || v == "" {
			return invalidSID
		}
		value = v
	case uint64:
		if v == 0 {
			return invalidSID
		}
		value = fmt.Sprintf("%d", v)
	case int32:
		if v == 0 {
			return invalidSID
		}
		value = fmt.Sprintf("%d", v)
	case int:
		if v == 0 {
			return invalidSID
		}
		value = fmt.Sprintf("%d", v)
	case int64:
		if v == 0 {
			return invalidSID
		}
		value = fmt.Sprintf("%d", v)
	default:
		return invalidSID
	}

	// steam2
	if match2 := reSteam2.FindStringSubmatch(value); match2 != nil {
		return fromSteam2Strings(match2)
	} else if match3 := reSteam3.FindStringSubmatch(value); match3 != nil {
		return fromSteam3Strings(match3)
	}

	// uint64 version
	intVal, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return invalidSID
	}

	if intVal < BaseSID {
		return fromUInt64(intVal)
	}

	return fromAccountID(intVal)
}

func (t *SteamID) Equal(id SteamID) bool {
	return t.AccountID == id.AccountID && t.AccountType == id.AccountType && t.Instance == id.Instance && t.Universe == id.Universe
}

func (t *SteamID) String() string {
	return fmt.Sprintf("%d", t.Int64())
}

func (t *SteamID) Int64() int64 {
	return int64((uint64(t.Universe << 56)) | (uint64(t.AccountType) << 52) | (uint64(t.Instance) << 32) | uint64(t.AccountID)) //nolint:gosec
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

func (t SteamID) MarshalJSON() ([]byte, error) {
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

// MarshalText implements encoding.TextMarshaler which is used by the yaml package for marshalling.
func (t SteamID) MarshalText() ([]byte, error) {
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

func (t *SteamID) Scan(value interface{}) error {
	if value == nil {
		*t = SteamID{}
		return nil
	}

	switch input := value.(type) {
	case string:
		if input == "" {
			*t = SteamID{}
			return nil
		}
		sid := New(input)
		if sid.Valid() {
			*t = sid
			return nil
		}
	case int64:
		sid := New(input)
		if sid.Valid() {
			*t = sid
			return nil
		}
	}

	return ErrInvalidSID
}

func (t SteamID) Value() (driver.Value, error) {
	return t.Int64(), nil
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
	sid.AccountID = SID32(id) //nolint:gosec

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
	usingApiKey := apiKey != ""
	u := "https://steamcommunity.com/id/" + query + "/?xml=1"
	if usingApiKey {
		u = urlVanity + url.Values{"key": {apiKey}, "vanityurl": {query}}.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return invalidSID, errors.Join(err, ErrRequestCreate)
	}

	resp, errDo := httpClient.Do(req)
	if errDo != nil {
		return invalidSID, errors.Join(errDo, ErrResponsePerform)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var vanityResp vanityURLResponse
	if usingApiKey {
		if errUnmarshal := json.NewDecoder(resp.Body).Decode(&vanityResp); errUnmarshal != nil {
			return invalidSID, errors.Join(errUnmarshal, ErrDecodeSID)
		}
	} else {
		vanityResp.Response.Success = 1
		content, errRead := io.ReadAll(resp.Body)
		if errRead != nil {
			return invalidSID, errors.Join(errRead, ErrResponseBody)
		}
		steamIDTags := reProfileID.FindStringSubmatch(string(content))
		if len(steamIDTags) >= 2 {
			vanityResp.Response.SteamID = New(steamIDTags[1])
		} else {
			return invalidSID, ErrResponseBody
		}
	}

	if vanityResp.Response.Success != 1 {
		return invalidSID, fmt.Errorf("%w: %d", ErrInvalidStatusCode, vanityResp.Response.Success)
	}
	if !vanityResp.Response.SteamID.Valid() {
		return invalidSID, fmt.Errorf("%w: %s", ErrInvalidSID, vanityResp.Response.SteamID.String())
	}

	return vanityResp.Response.SteamID, nil
}

// Resolve tries to retrieve a SteamID from a profile URL.
//
// If an error occurs or the SteamID was unable to be resolved from the query
// then am error is returned.
// TODO try and resolve len(17) && len(9) failed conversions as vanity.
func Resolve(ctx context.Context, query string) (SteamID, error) {
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
