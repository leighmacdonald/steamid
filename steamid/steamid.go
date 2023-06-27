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
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

const (
	urlVanity    = "https://api.steampowered.com/ISteamUser/ResolveVanityURL/v0001/?"
	baseIDString = "76561197960265728"
	BaseSID      = uint64(76561197960265728)
	BaseGID      = uint64(103582791429521408)
)

var (
	httpClient    *http.Client //nolint:gochecknoglobals
	reGroupIDTags = regexp.MustCompile(`<groupID64>(\w+)</groupID64>`)
	reGroupURL    = regexp.MustCompile(`steamcommunity.com/groups/(\S+)/?`)
	apiKey        string //nolint:gochecknoglobals

	// BuildVersion is replaced at compile time with the current tag or revision.
	BuildVersion = "master" //nolint:gochecknoglobals

	// ErrNoAPIKey is returned for functions that require an API key to use when one has not been set.
	ErrNoAPIKey = errors.New("No steam web api key, to obtain one see: " +
		"https://steamcommunity.com/dev/apikey and call steamid.SetKey()")
	ErrInvalidSID = errors.New("Invalid steam id")
)

// AppID is the id associated with games/apps.
type AppID uint32

// SID represents a SteamID
// STEAM_0:0:86173181.
type SID string

// SID64 represents a Steam64
// 76561198132612090.
type SID64 struct {
	big.Int
}

func New(value any) SID64 {
	s := SID64{}

	switch v := value.(type) {
	case string:
		parsedSid, errSid := StringToSID64(v)
		if errSid != nil {
			return SID64{}
		}

		s.SetInt64(parsedSid.Int64())
	case uint64:
		s.SetUint64(v)
	case int:
		s.SetInt64(int64(v))
	case int64:
		s.SetInt64(v)
	case float64:
		s.SetInt64(int64(v))
	default:
		s.SetInt64(0)
	}

	return s
}

func NewGID(value any) GID {
	s := GID{}

	switch v := value.(type) {
	case string:
		parsedGID, errGID := StringToSID64(v)
		if errGID != nil {
			return GID{}
		}

		s.SetInt64(parsedGID.Int64())
	case uint64:
		s.SetUint64(v)
	case int:
		s.SetInt64(int64(v))
	case int64:
		s.SetInt64(v)
	case float64:
		s.SetInt64(int64(v))
	default:
		s.SetInt64(0)
	}

	return s
}

// SID32 represents a Steam32
// 172346362.
type SID32 uint32

// SID3 represents a Steam3
// [U:1:172346362].
type SID3 string

// GID represents a GroupID (64bit)
// 103582791453729676.
type GID struct {
	big.Int
}

type Collection []SID64

func (c Collection) ToStringSlice() []string {
	var s []string

	for _, st := range c {
		s = append(s, st.String())
	}

	return s
}

func (c Collection) Contains(sid64 SID64) bool {
	for _, player := range c {
		if player.Uint64() == sid64.Uint64() {
			return true
		}
	}

	return false
}

// Valid ensures the value is at least large enough to be valid
// No further validation is done.
func (t SID64) Valid() bool {
	return t.Uint64() > BaseSID
}

func (t SID64) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%d\"", t.Int64())), nil
}

// UnmarshalJSON implements the Unmarshaler interface for steam ids. It will attempt to
// do all steam id types by calling StringToSID64.
func (t *SID64) UnmarshalJSON(data []byte) error {
	var (
		sidInput  any
		outputSid SID64
		err       error
	)

	if err = json.Unmarshal(data, &sidInput); err != nil {
		return errors.Wrapf(err, "failed to decode steamid: %v", err)
	}

	switch sid := sidInput.(type) {
	case string:
		outputSid, err = StringToSID64(sid)
		if err != nil {
			return errors.Wrap(err, "Failed to marshal string to SID64")
		}

		t.SetInt64(outputSid.Int64())
	case int64:
		t.SetInt64(sid)
	case float64:
		var z big.Int
		_, ok := z.SetString(sidInput.(string), 10)

		if !ok {
			return ErrInvalidSID
		}

		t.SetUint64(z.Uint64())
	default:
		return ErrInvalidSID
	}

	if !outputSid.Valid() {
		return ErrInvalidSID
	}

	return nil
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
		return errors.New("Tried to set invalid key, must be 32 chars or 0 to remove it")
	}

	apiKey = key

	return nil
}

var idGen = uint64(76561197960265728) //nolint:gochecknoglobals

// RandSID64 generates a unique random (numerically) valid steamid for testing.
func RandSID64() SID64 {
	return New(atomic.AddUint64(&idGen, 1))
}

// SID64FromString will attempt to convert a Steam64 formatted string into a SID64.
func SID64FromString(steamID string) (SID64, error) {
	if steamID == "" {
		return SID64{}, errors.Wrap(ErrInvalidSID, "Cannot convert empty string")
	}

	i, err := strconv.ParseInt(steamID, 10, 64)
	if err != nil {
		return SID64{}, errors.Wrap(err, "Failed to parse integer")
	}

	sid := New(i)
	if !sid.Valid() {
		return SID64{}, errors.Errorf("Invalid steam64 value")
	}

	return sid, nil
}

// GIDFromString will attempt to convert a properly formatted string to a GID.
func GIDFromString(steamID string) (GID, error) {
	if steamID == "" {
		return GID{}, errors.Errorf("Cannot convert empty string")
	}

	i, err := strconv.ParseInt(steamID, 10, 64)
	if err != nil {
		return GID{}, errors.Wrap(err, "Failed to parse integer from string")
	}

	g := GID{}
	g.SetInt64(i)

	if !g.Valid() {
		return GID{}, ErrInvalidSID
	}

	return g, nil
}

// Valid checks if the valid meets the minimum requirements to be considered valid.
func (t GID) Valid() bool {
	return t.Uint64() > BaseGID
}

// SIDToSID64 converts a given SteamID to a SID64.
// e.g. STEAM_0:0:86173181 -> 76561198132612090
//
// 0 is returned if the process was unsuccessful.
func SIDToSID64(steamID SID) SID64 {
	idParts := strings.Split(string(steamID), ":")
	magic, _ := new(big.Int).SetString(baseIDString, 10)
	steam64, _ := new(big.Int).SetString(idParts[2], 10)
	steam64 = steam64.Mul(steam64, big.NewInt(2))
	steam64 = steam64.Add(steam64, magic)
	auth, _ := new(big.Int).SetString(idParts[1], 10)

	return New(steam64.Add(steam64, auth).Int64())
}

// SIDToSID32 converts a given SteamID to a SID32.
// e.g. STEAM_0:0:86173181 -> 172346362
//
// 0 is returned if the process was unsuccessful.
func SIDToSID32(steamID SID) SID32 {
	return SID64ToSID32(SIDToSID64(steamID))
}

// SIDToSID3 converts a given SteamID to a SID3.
// e.g. STEAM_0:0:86173181 -> [U:1:172346362]
//
// An empty SID3 (string) is returned if the process was unsuccessful.
func SIDToSID3(steamID SID) SID3 {
	steamIDParts := strings.Split(string(steamID), ":")
	steamLastPart, errLast := strconv.ParseUint(steamIDParts[len(steamIDParts)-1], 10, 64)

	if errLast != nil {
		return ""
	}

	steamMidPart, errMid := strconv.ParseUint(steamIDParts[len(steamIDParts)-2], 10, 64)
	if errMid != nil {
		return ""
	}

	return SID3("[U:1:" + strconv.FormatUint((steamLastPart*2)+steamMidPart, 10) + "]")
}

// SID64ToSID converts a given SID64 to a SteamID.
// e.g. 76561198132612090 -> STEAM_0:0:86173181
//
// An empty SteamID (string) is returned if the process was unsuccessful.
func SID64ToSID(steam64 SID64) SID {
	steamID := new(big.Int).SetInt64(steam64.Int64())
	magic := new(big.Int).SetUint64(BaseSID)
	steamID = steamID.Sub(steamID, magic)
	isServer := new(big.Int).And(steamID, big.NewInt(1))
	steamID = steamID.Sub(steamID, isServer)
	steamID = steamID.Div(steamID, big.NewInt(2))

	return SID("STEAM_0:" + isServer.String() + ":" + steamID.String())
}

// SID64ToSID32 converts a given SID64 to a SID32.
// e.g. 76561198132612090 -> 172346362
//
// 0 is returned if the process was unsuccessful.
func SID64ToSID32(steam64 SID64) SID32 {
	steam64Str := strconv.FormatUint(steam64.Uint64(), 10)
	if len(steam64Str) < 3 {
		return 0
	}

	steam32, err := strconv.ParseInt(steam64Str[3:], 10, 64)
	if err != nil {
		return 0
	}

	return SID32(steam32 - 61197960265728)
}

// SID64ToSID3 converts a given SID64 to a SID3.
// e.g. 76561198132612090 -> [U:1:172346362]
//
// An empty SID3 (string) is returned if the process was unsuccessful.
func SID64ToSID3(steam64 SID64) SID3 {
	steamID := SID64ToSID(steam64)
	empty := New(0)

	if string(steamID) == empty.String() {
		return ""
	}

	return SIDToSID3(steamID)
}

// SID32ToSID converts a given SID32 to a SteamID.
// eg. 172346362 -> STEAM_0:0:86173181
//
// An empty SteamID (string) is returned if the process was unsuccessful.
func SID32ToSID(steam32 SID32) SID {
	return SID64ToSID(SID32ToSID64(steam32))
}

// SID32ToSID64 converts a given SID32 to a SID64.
// e.g. 172346362 -> 76561198132612090
//
// 0 is returned if the process was unsuccessful.
func SID32ToSID64(steam32 SID32) SID64 {
	steam64, err := strconv.ParseInt("765"+strconv.FormatInt(int64(steam32)+61197960265728, 10), 10, 64)
	if err != nil {
		return SID64{}
	}

	return New(steam64)
}

// SID32ToSID3 converts a given SID32 to a SID3.
// eg. 172346362 -> [U:1:172346362]
//
// An empty SID3 (string) is returned if the process was unsuccessful.
func SID32ToSID3(steam32 SID32) SID3 {
	steamID := SID32ToSID(steam32)
	if steamID == SID32ToSID(0) {
		return ""
	}

	return SIDToSID3(steamID)
}

// SID3ToSID converts a given SID3 to a SteamID.
// eg. [U:1:172346362] -> STEAM_0:0:86173181
//
// An empty SteamID (string) is returned if the process was unsuccessful.
func SID3ToSID(steam3 SID3) SID {
	parts := strings.Split(string(steam3), ":")
	id32 := parts[len(parts)-1]

	if len(id32) == 0 {
		return ""
	}

	if id32[len(id32)-1:] == "]" {
		id32 = id32[:len(id32)-1]
	}

	steam32, err := strconv.ParseUint(id32, 10, 64)
	if err != nil {
		return ""
	}

	return SID32ToSID(SID32(steam32))
}

// SID3ToSID64 converts a given SID3 to a SID64.
// eg. [U:1:172346362] -> 76561198132612090
//
// 0 is returned if the process was unsuccessful.
func SID3ToSID64(steam3 SID3) SID64 {
	parts := strings.Split(string(steam3), ":")
	id32 := parts[len(parts)-1]

	if len(id32) == 0 {
		return SID64{}
	}

	if id32[len(id32)-1:] == "]" {
		id32 = id32[:len(id32)-1]
	}

	steam32, err := strconv.ParseUint(id32, 10, 64)
	if err != nil {
		return SID64{}
	}

	return SID32ToSID64(SID32(steam32))
}

// SID3ToSID32 converts a given SID3 to a SID64.
// eg. [U:1:172346362] -> 172346362
//
// 0 is returned if the process was unsuccessful.
func SID3ToSID32(steam3 SID3) SID32 {
	parts := strings.Split(string(steam3), ":")
	id32 := parts[len(parts)-1]

	if len(id32) == 0 {
		return SID32(0)
	}

	if id32[len(id32)-1:] == "]" {
		id32 = id32[:len(id32)-1]
	}

	steam32, err := strconv.ParseUint(id32, 10, 64)
	if err != nil {
		return SID32(0)
	}

	return SID32(steam32)
}

// ResolveGID tries to resolve the GroupID from a group custom URL.
// NOTE This may be prone to error due to not being a real api endpoint.
func ResolveGID(ctx context.Context, groupVanityURL string) (GID, error) {
	m := reGroupURL.FindStringSubmatch(groupVanityURL)
	if len(m) > 0 {
		groupVanityURL = m[1]
	}

	u := "https://steamcommunity.com/groups/" + groupVanityURL + "/memberslistxml?xml=1"

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if errReq != nil {
		return GID{}, errors.Wrap(errReq, "Failed to create request")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return GID{}, errors.Wrapf(err, "Failed to fetch GID from host")
	}

	content, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return GID{}, errors.Wrapf(errRead, "Failed to read response body")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	groupIDTags := reGroupIDTags.FindSubmatch(content)
	if len(groupIDTags) >= 2 {
		groupID, errUint := strconv.ParseUint(string(groupIDTags[1]), 10, 64)
		if errUint != nil {
			return GID{}, errors.Wrapf(errUint, "Failed to convert GID to int")
		}

		g := GID{}
		g.SetUint64(groupID)

		if !g.Valid() {
			return GID{}, ErrInvalidSID
		}

		return g, nil
	}

	return GID{}, errors.Errorf("Failed to resolve GID: %s", content)
}

type vanityURLResponse struct {
	Response struct {
		SteamID string `json:"steamid"`
		Success int    `json:"success"`
	} `json:"response"`
}

// ResolveVanity attempts to resolve the underlying SID64 of a users vanity url name
// This only accepts the name or last portion of the /id/ profile link
// For https://steamcommunity.com/id/SQUIRRELLY the value is SQUIRRELLY.
func ResolveVanity(ctx context.Context, query string) (SID64, error) {
	if apiKey == "" {
		return SID64{}, ErrNoAPIKey
	}

	u := urlVanity + url.Values{"key": {apiKey}, "vanityurl": {query}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return SID64{}, errors.Wrap(err, "Failed to create request")
	}

	resp, errDo := httpClient.Do(req)
	if errDo != nil {
		return SID64{}, errors.Wrapf(errDo, "Failed to perform vanity lookup")
	}

	content, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return SID64{}, errors.Wrapf(errRead, "Failed to read vanity response body")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var vanityResp vanityURLResponse
	if errUnmarshal := json.Unmarshal(content, &vanityResp); err != nil {
		return SID64{}, errors.Wrap(errUnmarshal, "Failed to decode json vanity response")
	}

	if vanityResp.Response.Success != 1 {
		return SID64{}, errors.Errorf("Invalid success code received: %d", vanityResp.Response.Success)
	}

	if len(vanityResp.Response.SteamID) != 17 {
		return SID64{}, errors.Errorf("Malformed steamid received: %s", vanityResp.Response.SteamID)
	}

	output, errParseInt := strconv.ParseInt(vanityResp.Response.SteamID, 10, 64)
	if errParseInt != nil {
		return SID64{}, errors.Wrap(err, "Failed to parse int from steamid received")
	}

	sidOut := SID64{}
	sidOut.SetInt64(output)

	return sidOut, nil
}

// ResolveSID64 tries to retrieve a SteamID64 using a query (search).
//
// If an error occurs or the SteamID was unable to be resolved from the query
// then am error is returned.
// TODO try and resolve len(17) && len(9) failed conversions as vanity.
func ResolveSID64(ctx context.Context, query string) (SID64, error) {
	query = strings.ReplaceAll(query, " ", "")
	if strings.Contains(query, "steamcommunity.com/profiles/") {
		if string(query[len(query)-1]) == "/" {
			query = query[0 : len(query)-1]
		}

		output, err := strconv.ParseInt(query[strings.Index(query, "steamcommunity.com/profiles/")+len("steamcommunity.com/profiles/"):], 10, 64)
		if err != nil {
			return SID64{}, errors.Wrapf(err, "Failed to parse int from query")
		}

		// query = strings.Replace(query, "/", "", -1)
		if len(strconv.FormatInt(output, 10)) != 17 {
			return SID64{}, errors.Wrapf(err, "Invalid string length")
		}

		return New(output), nil
	} else if strings.Contains(query, "steamcommunity.com/id/") {
		if string(query[len(query)-1]) == "/" {
			query = query[0 : len(query)-1]
		}
		query = query[strings.Index(query, "steamcommunity.com/id/")+len("steamcommunity.com/id/"):]
		return ResolveVanity(ctx, query)
	}

	s, err := StringToSID64(query)
	if err == nil {
		return s, nil
	}

	return ResolveVanity(ctx, query)
}

// StringToSID64 will attempt to convert a string containing some format of steam id into
// a SID64 automatically, picking the appropriate matching conversion to make.
//
//	This will not resolve vanity ids. Use ResolveSID64 if you also want to attempt
//
// to resolve it as a vanity url in addition.
func StringToSID64(s string) (SID64, error) {
	us := strings.ToUpper(s)

	if len(s) == 17 {
		i64, err := strconv.ParseUint(s, 10, 64)
		if err == nil {
			v := SID64{}
			v.SetUint64(i64)

			if !v.Valid() {
				return SID64{}, errors.Errorf("String provided is not a valid Steam64 format: %s", s)
			}

			return v, nil
		}
	}

	if len(s) == 9 {
		i32, err := strconv.ParseUint(s, 10, 32)
		if err == nil {
			v := SID32ToSID64(SID32(i32))
			if !v.Valid() {
				return SID64{}, errors.Errorf("String provided is not a valid Steam32 format: %s", s)
			}

			return v, nil
		}
	}

	if strings.HasPrefix(us, "[U:") {
		v := SID3ToSID64(SID3(us))
		if !v.Valid() {
			return SID64{}, errors.Errorf("String provided is not a valid Steam3 format: %s", s)
		}

		return v, nil
	}

	if strings.HasPrefix(us, "STEAM_") {
		v := SIDToSID64(SID(us))
		if !v.Valid() {
			return v, errors.Errorf("String provided is not a valid Steam format: %s", s)
		}

		return v, nil
	}

	return SID64{}, errors.Errorf("String provided did not match any know steam formats: %s", s)
}

// ParseString attempts to parse any strings of any known format within the body to a common SID64 format.
func ParseString(body string) []SID64 {
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

	var ids []SID64
	for k := range found {
		ids = append(ids, New(k))
	}

	return ids
}

func init() {
	if t, found := os.LookupEnv("STEAM_TOKEN"); found && t != "" {
		if err := SetKey(t); err != nil {
			panic(err)
		}
	}

	httpClient = &http.Client{
		Timeout: time.Second * 10,
	}
}
