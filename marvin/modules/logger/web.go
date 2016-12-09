package logger

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/modules/weblogin"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/slack/rtm"
	"github.com/riking/homeapi/marvin/util"
)

func (mod *LoggerModule) getPrivateChannels(userID slack.UserID, w http.ResponseWriter, r *http.Request) ([]briefChannelInfo, error) {
	item, ok := mod.cache.Get(fmt.Sprintf("groups-%s", userID))
	if ok {
		return item.([]briefChannelInfo), nil
	}

	token, err := mod.team.GetModule(weblogin.Identifier).(weblogin.API).GetUserToken(w, r)
	if err != nil {
		return nil, err
	}
	form := url.Values{
		"token": []string{token},
	}
	var resp struct {
		Groups []*slack.Channel `json:"groups"`
	}
	err = mod.team.SlackAPIPostJSON("groups.list", form, &resp)
	if err != nil {
		return nil, err
	}
	yourChannels := make([]briefChannelInfo, 0, len(resp.Groups))
	for i, v := range resp.Groups {
		if v.IsMultiIM() {
			continue
		}
		yourChannels = append(yourChannels, briefChannelInfo{
			Name:    v.Name,
			ID:      v.ID,
			Purpose: v.Purpose,
			Members: make([]struct{}, len(v.Members)),
		})
		g, _ := mod.team.PrivateChannelInfo(v.ID)
		if g != nil {
			yourChannels[i].HasMarvin = true
		}
	}

	mod.cache.SetDefault(fmt.Sprintf("groups-%s", userID), yourChannels)
	return yourChannels, nil
}

var tmplIndex = template.Must(weblogin.LayoutTemplateCopy().Parse(string(weblogin.MustAsset("templates/logs-index.html"))))

type briefChannelInfo struct {
	Name      string
	ID        slack.ChannelID
	HasMarvin bool `json:"-"`
	Purpose   slack.ChannelTopicPurpose
	Members   []struct{}
}

func (mod *LoggerModule) LogsIndex(w http.ResponseWriter, r *http.Request) {
	lc, err := weblogin.NewLayoutContent(mod.team, w, r, weblogin.NavSectionLogs)
	if err != nil {
		mod.team.GetModule(weblogin.Identifier).(weblogin.API).HTTPError(w, r, err)
		return
	}

	var data struct {
		Team                marvin.Team
		Channels            []*slack.Channel
		YourPrivateChannels []briefChannelInfo
	}
	data.Team = mod.team

	// Fill out YourPrivateChannels
	if lc.CurrentUser != nil {
		data.YourPrivateChannels, err = mod.getPrivateChannels(lc.CurrentUser.ID, w, r)
		if err != nil {
			mod.team.GetModule(weblogin.Identifier).(weblogin.API).HTTPError(w, r, err)
			return
		}
	}

	rtmClient := mod.team.GetRTMClient().(*rtm.Client)
	data.Channels = rtmClient.Channels
	lc.BodyData = data

	util.LogIfError(tmplIndex.ExecuteTemplate(w, "layout", lc))
}
