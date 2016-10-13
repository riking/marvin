package slack

import (
	"net/url"

	"encoding/json"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/shocky"
	"github.com/riking/homeapi/shocky/slack"
	"golang.org/x/net/websocket"
)

type Client struct {
	conn websocket.Conn
	team shocky.Team

	Self struct {
		ID             slack.UserID
		Name           string
		Prefs          map[string]interface{}
		Created        float64
		ManualPresence string `json:"manual_presence"`
	}
	Users     []slack.User
	AboutTeam slack.TeamInfo
	Channels  []slack.Channel
	Groups    []slack.Channel
	Mpims     []slack.Channel
	Ims       []slack.Channel
}

const startAPIURL = "https://slack.com/api/rtm.start"

func Dial(team shocky.Team) (*Client, error) {
	data := url.Values{}
	data.Set("token", team.TeamConfig().UserToken)
	data.Set("no-unreads", true)
	data.Set("mipm-aware", true)
	var startResponse struct {
		slack.APIResponse
		URL string
		*Client
	}
	resp, err := team.HTTPClient().PostForm(startAPIURL, data)
	if err != nil {
		return errors.Wrap(err, "slack post rtm.start")
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "slack post rtm.start read body")
	}
	err = json.Unmarshal(respBytes, &startResponse)
	if err != nil {
		return errors.Wrap(err, "slack post rtm.start unmarshal")
	}
	if !startResponse.OK {
		return errors.Wrap(startResponse.APIResponse, "slack post rtm.start error")
	}
}
