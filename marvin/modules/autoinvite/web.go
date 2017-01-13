package autoinvite

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin/modules/weblogin"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

func (mod *AutoInviteModule) registerHTTP() {
	mod.team.Router().HandleFunc("/invites", mod.HTTPListInvites)
	mod.team.Router().Path("/invites/{channel}").Methods(http.MethodPost).HandlerFunc(mod.HTTPInvite)
}

var tmplListInvites = template.Must(weblogin.LayoutTemplateCopy().Parse(string(weblogin.MustAsset("templates/invite-list.html"))))

func (mod *AutoInviteModule) HTTPListInvites(w http.ResponseWriter, r *http.Request) {
	wlAPI := mod.team.GetModule(weblogin.Identifier).(weblogin.API)

	user, err := wlAPI.GetCurrentUser(w, r)
	if err != nil {
		wlAPI.HTTPError(w, r, errors.Wrap(err, "Error determining login state"))
		return
	}

	lc, _ := weblogin.NewLayoutContent(mod.team, w, r, weblogin.NavSectionInvite)

	stmt, err := mod.team.DB().Prepare(sqlListInvites)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type singleChannel struct {
		Name string
		ID   slack.ChannelID

		Available bool

		User        slack.UserID
		UserName    string
		Timestamp   time.Time
		Text        string
		MemberCount int
		Purpose     string
	}

	var data struct {
		Layout         *weblogin.LayoutContent
		NotLoggedIn    bool
		NeedPermission bool
		HaveJoinData   bool

		Channels []singleChannel
	}
	data.Layout = lc
	data.NotLoggedIn = user == nil || user.SlackUser == ""

	rows, err := stmt.Query()
	if err != nil {
		wlAPI.HTTPError(w, r, errors.Wrap(err, "Database query error"))
		return
	}

	seenChannels := make(map[slack.ChannelID]bool)

	for rows.Next() {
		var inviteChannelStr, inviteUserStr, inviteTS, inviteText string
		err = rows.Scan(&inviteChannelStr, &inviteUserStr, &inviteTS, &inviteText)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		inviteChannelID := slack.ChannelID(inviteChannelStr)
		if seenChannels[inviteChannelID] {
			continue
		}
		seenChannels[inviteChannelID] = true

		idx := strings.IndexByte(inviteTS, '.')
		inviteUnix, _ := strconv.ParseInt(inviteTS[:idx], 10, 64)
		inviteTime := time.Unix(inviteUnix, 0)
		inviteChannelName := mod.team.ChannelName(slack.ChannelID(inviteChannelStr))
		channelInfo, _ := mod.team.PrivateChannelInfo(inviteChannelID)

		data.Channels = append(data.Channels, singleChannel{
			ID:          inviteChannelID,
			Name:        inviteChannelName,
			Available:   false,
			User:        slack.UserID(inviteUserStr),
			UserName:    mod.team.UserName(slack.UserID(inviteUserStr)),
			Timestamp:   inviteTime,
			Text:        inviteText,
			MemberCount: mod.team.ChannelMemberCount(inviteChannelID),
			Purpose:     channelInfo.Purpose.Value,
		})
	}

	data.HaveJoinData = false
	if user != nil && user.SlackUser != "" {
		slackChannels := make([]slack.ChannelID, len(data.Channels))
		for i, v := range data.Channels {
			slackChannels[i] = v.ID
		}
		membershipMap := mod.team.UserInChannels(user.SlackUser, slackChannels...)
		for i := range data.Channels {
			data.Channels[i].Available = !membershipMap[data.Channels[i].ID]
		}
		data.HaveJoinData = true
	}

	lc.BodyData = data
	util.LogIfError(
		tmplListInvites.Execute(w, lc))
}

var rgxAcceptInvite = regexp.MustCompile(`/invites/([A-Z0-9]+)`)

type jsonResponse struct {
	OK    bool `json:"ok"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Data interface{} `json:"data"`
}

func (mod *AutoInviteModule) HTTPInvite(w http.ResponseWriter, r *http.Request) {
	wlAPI := mod.team.GetModule(weblogin.Identifier).(weblogin.API)

	user, err := wlAPI.GetCurrentUser(w, r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"ok":false,"error":{"type":"bad_cookie","message":"Bad cookie."}`)
		return
	}

	if user == nil || user.SlackUser == "" {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"ok":false,"error":{"type":"not_logged_in","message":"You are not logged in."}`)
		return
	}

	m := rgxAcceptInvite.FindStringSubmatch(r.URL.Path)
	if m == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ok":false,"error":{"type":"bad_url","message":"Channel ID not found."}`)
		return
	}

	channelID := m[1]

	var response struct {
		AlreadyInGroup bool `json:"already_in_group"`
	}
	form := url.Values{
		"channel": []string{channelID},
		"user":    []string{string(user.SlackUser)},
	}
	err = mod.team.SlackAPIPostJSON("groups.invite", form, &response)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(jsonResponse{
			OK: false,
			Error: struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}{
				Type:    "slack",
				Message: fmt.Sprintf("slack reported an error: %s", err),
			},
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(jsonResponse{
		OK: true,
		Data: struct {
			AlreadyJoined bool `json:"already_joined"`
		}{
			AlreadyJoined: response.AlreadyInGroup,
		},
	})
}
