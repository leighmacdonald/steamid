// Package steamid provides conversion to and from all steam ID formats.
//
// If you wish to resolve vanity names like https://steamcommunity.com/id/SQUIRRELLY into
// steam id, or to GetPlayerSummaries requests against the WebAPI you must first obtain a
// API key at https://steamcommunity.com/dev/apikey.
//
// You can then set it for the package to use:
//
// 		steamid.SetKey(apiKey)
//
//	With a steam api key set you can now use the following functions:
//
//		steamid.ResolveVanity()
//
//
package steamid

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	urlVanity = "https://api.steampowered.com/ISteamUser/ResolveVanityURL/v0001/?"
)

var (
	httpClient    *http.Client
	reGroupIDTags *regexp.Regexp
	reGroupURL    *regexp.Regexp
	apiKey        string

	// BuildVersion is replaced at compile time with the current tag or revision
	BuildVersion = "master"

	// ErrNoAPIKey is returned for functions that require an API key to use when one has not been set
	ErrNoAPIKey = errors.New("No steam web api key, to obtain one see: " +
		"https://steamcommunity.com/dev/apikey and call steamid.SetKey()")
)

// AppID is the Steam appdb ID
type AppID int64

// SID represents a SteamID
// STEAM_0:0:86173181
type SID string

// SID64 represents a Steam64
// 76561198132612090
type SID64 uint64

// SID32 represents a Steam32
// 172346362
type SID32 uint32

// SID3 represents a Steam3
// [U:1:172346362]
type SID3 string

// GID represents a GroupID (64bit)
// 103582791453729676
type GID uint64 // 103582791453729676

// String renders the GID as a int64 string
func (t GID) String() string {
	return strconv.FormatInt(int64(t), 10)
}

// String renders the SID64 as a int64 string
func (t SID64) String() string {
	return strconv.FormatInt(int64(t), 10)
}

// Int64 casts the strings to a raw int64
func (t SID64) Int64() int64 {
	return int64(t)
}

// Valid ensures the value is at least large enough to be valid
// No further validation is done.
func (t SID64) Valid() bool {
	return t > 76561197960265728
}

// UnmarshalJSON implements the Unmarshaler interface for steam ids. It will attempt to
// do all steam id types by calling StringToSID64
func (t *SID64) UnmarshalJSON(data []byte) error {
	var sid string
	if err := json.Unmarshal(data, &sid); err != nil {
		return errors.Wrapf(err, "failed to decode steamid: %v", err)
	}
	v, err := StringToSID64(sid)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal string to SID64")
	}
	if !v.Valid() {
		return errors.Errorf("Invalid steam id: %s", sid)
	}
	*t = v
	return nil
}

// SetKey will set the package global steam webapi key used for some requests
// Basic id conversion usage does not require this to be set.
//
// You can alternatively set the key with the environment variable `STEAM_TOKEN={YOUR_API_KEY`
// To get a key see: https://steamcommunity.com/dev/apikey
func SetKey(key string) {
	if len(key) != 32 && len(key) != 0 {
		log.Warnf("Tried to set invalid key, must be 32 chars or 0 to remove it")
		return
	}
	apiKey = key
}

// GetKey returns the steam web api key, if set, otherwise empty string
func GetKey() string {
	return apiKey
}

// GetHTTP returns the currently configured http.Client
func GetHTTP() *http.Client {
	return httpClient
}

var idGen uint64 = 76561197960265728

// RandSID64 generates a unique random (numerically) valid steamid for testing
func RandSID64() SID64 {
	return SID64(atomic.AddUint64(&idGen, 1))
}

// SID64FromString will attempt to convert a Steam64 formatted string into a SID64
func SID64FromString(steamID string) (SID64, error) {
	if steamID == "" {
		return 0, errors.New("Cannot convert empty string")
	}
	i, err := strconv.ParseInt(steamID, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to parse integer")
	}
	sid := SID64(i)
	if !sid.Valid() {
		return 0, errors.Errorf("Invalid steam64 value")
	}
	return sid, nil
}

// GIDFromString will attempt to convert a properly formatted string to a GID
func GIDFromString(steamID string) (GID, error) {
	if steamID == "" {
		return 0, errors.Errorf("Cannot convert empty string")
	}
	i, err := strconv.ParseInt(steamID, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to parse integer from string")
	}
	return GID(i), nil
}

// Valid checks if the valid meets the minimum requirements to be considered valid
func (t GID) Valid() bool {
	return t > 103582791429521408
}

// Int64 just cast the value ti a bare int64
func (t GID) Int64() int64 {
	return int64(t)
}

// SIDToSID64 converts a given SteamID to a SID64.
// eg. STEAM_0:0:86173181 -> 76561198132612090
//
// 0 is returned if the process was unsuccessful.
func SIDToSID64(steamID SID) SID64 {
	idParts := strings.Split(string(steamID), ":")
	magic, _ := new(big.Int).SetString("76561197960265728", 10)
	steam64, _ := new(big.Int).SetString(idParts[2], 10)
	steam64 = steam64.Mul(steam64, big.NewInt(2))
	steam64 = steam64.Add(steam64, magic)
	auth, _ := new(big.Int).SetString(idParts[1], 10)
	return SID64(steam64.Add(steam64, auth).Int64())
}

// SIDToSID32 converts a given SteamID to a SID32.
// eg. STEAM_0:0:86173181 -> 172346362
//
// 0 is returned if the process was unsuccessful.
func SIDToSID32(steamID SID) SID32 {
	return SID64ToSID32(SIDToSID64(steamID))
}

// SIDToSID3 converts a given SteamID to a SID3.
// eg. STEAM_0:0:86173181 -> [U:1:172346362]
//
// An empty SID3 (string) is returned if the process was unsuccessful.
func SIDToSID3(steamID SID) SID3 {
	steamIDParts := strings.Split(string(steamID), ":")
	steamLastPart, err := strconv.ParseUint(steamIDParts[len(steamIDParts)-1], 10, 64)
	if err != nil {
		return ""
	}
	return SID3("[U:1:" + strconv.FormatUint(steamLastPart*2, 10) + "]")
}

// SID64ToSID converts a given SID64 to a SteamID.
// eg. 76561198132612090 -> STEAM_0:0:86173181
//
// An empty SteamID (string) is returned if the process was unsuccessful.
func SID64ToSID(steam64 SID64) SID {
	steamID := new(big.Int).SetInt64(int64(steam64))
	magic, _ := new(big.Int).SetString("76561197960265728", 10)
	steamID = steamID.Sub(steamID, magic)
	isServer := new(big.Int).And(steamID, big.NewInt(1))
	steamID = steamID.Sub(steamID, isServer)
	steamID = steamID.Div(steamID, big.NewInt(2))
	return SID("STEAM_0:" + isServer.String() + ":" + steamID.String())
}

// SID64ToSID32 converts a given SID64 to a SID32.
// eg. 76561198132612090 -> 172346362
//
// 0 is returned if the process was unsuccessful.
func SID64ToSID32(steam64 SID64) SID32 {
	steam64Str := strconv.FormatUint(uint64(steam64), 10)
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
// eg. 76561198132612090 -> [U:1:172346362]
//
// An empty SID3 (string) is returned if the process was unsuccessful.
func SID64ToSID3(steam64 SID64) SID3 {
	steamID := SID64ToSID(steam64)
	if steamID == SID64ToSID(0) {
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
// eg. 172346362 -> 76561198132612090
//
// 0 is returned if the process was unsuccessful.
func SID32ToSID64(steam32 SID32) SID64 {
	steam64, err := strconv.ParseInt("765"+strconv.FormatInt(int64(steam32)+61197960265728, 10), 10, 64)
	if err != nil {
		return 0
	}
	return SID64(steam64)
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
	if len(id32) <= 0 {
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
	if len(id32) <= 0 {
		return SID64(0)
	}
	if id32[len(id32)-1:] == "]" {
		id32 = id32[:len(id32)-1]
	}
	steam32, err := strconv.ParseUint(id32, 10, 64)
	if err != nil {
		return SID64(0)
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
	if len(id32) <= 0 {
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
// NOTE This may be prone to error due to not being a real api endpoint
func ResolveGID(ctx context.Context, groupVanityURL string) (GID, error) {
	m := reGroupURL.FindStringSubmatch(groupVanityURL)
	if len(m) > 0 {
		groupVanityURL = m[1]
	}
	u := "https://steamcommunity.com/groups/" + groupVanityURL + "/memberslistxml?xml=1"
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to create request")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return GID(0), errors.Wrapf(err, "Failed to fetch GID from host")
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return GID(0), errors.Wrapf(err, "Failed to read response body")
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	groupIDTags := reGroupIDTags.FindSubmatch(content)
	if len(groupIDTags) >= 2 {
		groupID, err := strconv.ParseUint(string(groupIDTags[1]), 10, 64)
		if err != nil {
			return GID(0), errors.Wrapf(err, "Failed to convert GID to int")
		}
		return GID(groupID), nil
	}
	return GID(0), errors.Errorf("Failed to resolve GID: %s", content)
}

// ResolveVanity attempts to resolve the underlying SID64 of a users vanity url name
// This only accepts the name or last portion of the /id/ profile link
// For https://steamcommunity.com/id/SQUIRRELLY the value is SQUIRRELLY
func ResolveVanity(ctx context.Context, query string) (SID64, error) {
	if apiKey == "" {
		return 0, ErrNoAPIKey
	}
	u := urlVanity + url.Values{"key": {apiKey}, "vanityurl": {query}}.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to create request")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return SID64(0), errors.Wrapf(err, "Failed to perform vanity lookup")
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return SID64(0), errors.Wrapf(err, "Failed to read vanity response body")
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	var vanityURLResponse struct {
		Response struct {
			Steamid string
			Success int
		}
	}
	if err := json.Unmarshal(content, &vanityURLResponse); err != nil {
		return SID64(0), errors.Wrap(err, "Failed to decode json vanity response")
	}
	if vanityURLResponse.Response.Success != 1 {
		return SID64(0), errors.Errorf("Invalid success code received: %d", vanityURLResponse.Response.Success)
	}
	if len(vanityURLResponse.Response.Steamid) != 17 {
		return SID64(0), errors.Errorf("Malformed steamid received: %s", vanityURLResponse.Response.Steamid)
	}
	output, err := strconv.ParseInt(vanityURLResponse.Response.Steamid, 10, 64)
	if err != nil {
		return SID64(0), errors.Wrap(err, "Failed to parse int from steamid received")
	}
	return SID64(output), nil
}

// ResolveSID64 tries to retrieve a SteamID64 using a query (search).
//
// If an error occurs or the SteamID was unable to be resolved from the query
// then am error is returned.
// TODO try and resolve len(17) && len(9) failed conversions as vanity
func ResolveSID64(ctx context.Context, query string) (SID64, error) {
	query = strings.Replace(query, " ", "", -1)
	if strings.Contains(query, "steamcommunity.com/profiles/") {
		if string(query[len(query)-1]) == "/" {
			query = query[0 : len(query)-1]
		}
		output, err := strconv.ParseInt(query[strings.Index(query, "steamcommunity.com/profiles/")+len("steamcommunity.com/profiles/"):], 10, 64)
		if err != nil {
			return SID64(0), errors.Wrapf(err, "Failed to parse int from query")
		}
		//query = strings.Replace(query, "/", "", -1)
		if len(strconv.FormatInt(output, 10)) != 17 {
			return SID64(0), errors.Wrapf(err, "Invalid string length")
		}
		return SID64(output), nil
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
//  This will not resolve vanity ids. Use ResolveSID64 if you also want to attempt
// to resolve it as a vanity url in addition.
func StringToSID64(s string) (SID64, error) {
	us := strings.ToUpper(s)
	if len(s) == 17 {
		i64, err := strconv.ParseUint(s, 10, 64)
		if err == nil {
			v := SID64(i64)
			if !v.Valid() {
				return v, errors.Errorf("String provided is not a valid Steam64 format: %s", s)
			}
			return SID64(i64), nil
		}
	}
	if len(s) == 9 {
		i32, err := strconv.ParseUint(s, 10, 32)
		if err == nil {
			v := SID32ToSID64(SID32(i32))
			if !v.Valid() {
				return v, errors.Errorf("String provided is not a valid Steam32 format: %s", s)
			}
			return v, nil
		}
	}
	if strings.HasPrefix(us, "[U:") {
		v := SID3ToSID64(SID3(us))
		if !v.Valid() {
			return v, errors.Errorf("String provided is not a valid Steam3 format: %s", s)
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
	return 0, errors.Errorf("String provided did not match any know steam formats: %s", s)
}

// ParseString attempts to parse any strings of any known format within the body to a common SID64 format
func ParseString(body string) []SID64 {
	freSID := regexp.MustCompile(`STEAM_0:[01]:[0-9][0-9]{0,8}`)
	freSID64 := regexp.MustCompile(`7656119\d{10}`)
	freSID3 := regexp.MustCompile(`\[U:1:\d+]`)

	// Store only unique entries
	found := make(map[SID64]bool)
	m0 := freSID.FindAllStringSubmatch(body, -1)
	m1 := freSID64.FindAllStringSubmatch(body, -1)
	m2 := freSID3.FindAllStringSubmatch(body, -1)
	for _, i := range m0 {
		found[SIDToSID64(SID(i[0]))] = true
	}

	for _, i := range m1 {
		s, err := SID64FromString(i[0])
		if err != nil {
			log.Warnf("Failed to parse string to sid64: %s", i[0])
			continue
		}
		found[s] = true
	}

	for _, i := range m2 {
		found[SID3ToSID64(SID3(i[0]))] = true
	}
	var ids []SID64
	for k := range found {
		ids = append(ids, k)
	}
	return ids
}

func init() {
	if t, found := os.LookupEnv("STEAM_TOKEN"); found && t != "" {
		SetKey(t)
	}
	httpClient = &http.Client{
		Timeout: time.Second * 10,
	}
	reGroupIDTags = regexp.MustCompile(`<groupID64>(\w+)</groupID64>`)
	reGroupURL = regexp.MustCompile(`steamcommunity.com/groups/(\S+)/?`)
}
