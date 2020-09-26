package extra

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/steamid/steamid"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"strings"
)

// PlayerSummary is the unaltered player summary from the steam official API
type PlayerSummary struct {
	Steamid                  string `json:"steamid"`
	CommunityVisibilityState int    `json:"communityvisibilitystate"`
	ProfileState             int    `json:"profilestate"`
	PersonaName              string `json:"personaname"`
	ProfileURL               string `json:"profileurl"`
	Avatar                   string `json:"avatar"`
	AvatarMedium             string `json:"avatarmedium"`
	AvatarFull               string `json:"avatarfull"`
	AvatarHash               string `json:"avatarhash"`
	PersonaState             int    `json:"personastate"`
	RealName                 string `json:"realname"`
	PrimaryClanID            string `json:"primaryclanid"`
	TimeCreated              int    `json:"timecreated"`
	PersonastateFlags        int    `json:"personastateflags"`
	LocCountryCode           string `json:"loccountrycode"`
	LocStateCode             string `json:"locstatecode"`
	LocCityID                int    `json:"loccityid"`
}

type playerSummariesResp struct {
	Response struct {
		Players []PlayerSummary `json:"players"`
	} `json:"response"`
}

// PlayerSummaries will call GetPlayerSummaries on the valve WebAPI returning the players
// portion of the response as []PlayerSummary
//
// It will only accept up to 100 steamids in a single call
func PlayerSummaries(ctx context.Context, steamIDs []steamid.SID64) ([]PlayerSummary, error) {
	var ps []PlayerSummary
	apiKey := steamid.GetKey()
	if apiKey == "" {
		return ps, steamid.ErrNoAPIKey
	}
	const baseURL = "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=%s&steamids=%s"
	if len(steamIDs) == 0 {
		return ps, nil
	}
	if len(steamIDs) > 100 {
		return ps, errors.New("Too many steam ids, max 100")
	}
	var idStrings []string
	for _, id := range steamIDs {
		idStrings = append(idStrings, fmt.Sprintf("%d", id))
	}
	u := fmt.Sprintf(baseURL, apiKey, strings.Join(idStrings, ","))
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return ps, errors.Wrap(err, "Failed to create new request")
	}
	resp, err := steamid.GetHTTP().Do(req)
	if err != nil {
		return ps, errors.Wrap(err, "Failed to perform http request")
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ps, errors.Wrap(err, "Failed to read response body")
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	var r playerSummariesResp
	if err := json.Unmarshal(b, &r); err != nil {
		return ps, errors.Wrap(err, "Failed to decode JSON response")
	}
	return r.Response.Players, nil
}