package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"

	"github.com/pkg/errors"
	"gopkg.in/ini.v1"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/slack/controller"
	"github.com/riking/homeapi/marvin/slack/rtm"
	"github.com/riking/homeapi/marvin/util"

	_ "github.com/riking/homeapi/marvin/modules/_all"
)

func main() {
	teamName := flag.String("team", "Test", "which team to use")
	configFile := flag.String("conf", "", "override config file")
	flag.Parse()

	var cfg *ini.File
	var err error
	if *configFile != "" {
		cfg, err = ini.Load(*configFile)
	} else {
		cfg, err = ini.LooseLoad("testdata/config.ini", "config.ini", "/tank/www/apiserver/config.ini")
	}
	if err != nil {
		util.LogError(errors.Wrap(err, "loading config"))
		return
	}

	teamConfig := marvin.LoadTeamConfig(cfg.Section(*teamName))
	team, err := controller.NewTeam(teamConfig)
	if err != nil {
		util.LogError(errors.Wrap(err, "NewTeam"))
		return
	}
	l, err := net.Listen("tcp4", teamConfig.HTTPListen)
	if err != nil {
		util.LogError(errors.Wrap(err, "listen tcp"))
		return
	}
	client, err := rtm.Dial(team)
	if err != nil {
		util.LogError(errors.Wrap(err, "rtm.Dial"))
		return
	}
	client.RegisterRawHandler("main.go", func(msg slack.RTMRawMessage) {
	typeswitch:
		switch msg.Type() {
		case "user_typing", "reconnect_url", "presence_change":
			return
		case "pref_change":
			if msg.StringField("name") == "emoji_use" {
				return
			}
		case "message":
			switch msg.Subtype() {
			case "", "channel_leave", "channel_join", "group_join", "group_leave":
				break
			case "message_changed":
				type structMessage struct {
					Edited struct {
						TS   slack.MessageTS `json:"ts"`
						User slack.UserID    `json:"user"`
					}
					Text string
					TS   slack.MessageTS `json:"ts"`
					User slack.UserID    `json:"user"`
					Type string
				}
				var msgStruct struct {
					Channel     slack.ChannelID `json:"channel"`
					EventTS     slack.MessageTS `json:"event_ts"`
					Message     structMessage   `json:"message"`
					PrevMessage structMessage   `json:"previous_message"`
					Subtype     string          `json:"subtype"`
					Type        string          `json:"type"`
					TS          slack.MessageTS `json:"ts"`
				}
				json.Unmarshal(msg.Original(), &msgStruct)
				fmt.Printf("[%s] [EDIT BY %s] [@%s] %s\n", team.ChannelName(msgStruct.Channel), team.UserName(msgStruct.Message.Edited.User), team.UserName(msgStruct.Message.User), msgStruct.Message.Text)
				return
			default:
				break typeswitch
			}
			fmt.Printf("[%s] [@%s] %s\n", team.ChannelName(msg.ChannelID()), team.UserName(msg.UserID()), msg.Text())
			return
		case "reaction_added":
			item, ok := msg["item"].(map[string]interface{})
			if !ok {
				break
			}
			if item["type"].(string) != "message" {
				break
			}
			ts := slack.MessageTS(item["ts"].(string))
			channel := slack.ChannelID(item["channel"].(string))
			fmt.Printf("[%s] :%s: @%s -> @%s %s\n", team.ChannelName(channel),
				msg.StringField("reaction"), team.UserName(msg.UserID()),
				team.UserName(slack.UserID(msg.StringField("item_user"))), team.ArchiveURL(slack.MsgID(channel, ts)))
			return
		}
		util.LogDebug("main.go rtm message:", msg)
	}, rtm.MsgTypeAll, nil)

	team.ConnectRTM(client)
	team.EnableModules()
	team.ConnectHTTP(l)

	client.Start()

	fmt.Println("started")
	select {}
}
