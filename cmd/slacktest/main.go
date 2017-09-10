package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/slack/controller"
	"github.com/riking/marvin/slack/rtm"
	"github.com/riking/marvin/util"
	"gopkg.in/ini.v1"
)

import (
	_ "github.com/riking/marvin/modules/_all"
)

var colorDebug = ansi.ColorFunc("black+h")

func messagePrinter(team marvin.Team, dumpMessages bool) func(msg slack.RTMRawMessage) {
	return func(msg slack.RTMRawMessage) {
	typeswitch:
		switch msg.Type() {
		case "user_typing", "reconnect_url", "presence_change":
			return
		case "pref_change":
			if msg.StringField("name") == "emoji_use" {
				return
			}
		case "message":
		subtypeswitch:
			switch msg.Subtype() {
			case "", "channel_leave", "channel_join", "group_join", "group_leave":
				break subtypeswitch
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
				fmt.Printf("[%s%s] [EDIT BY %s] [@%s] %s\n", team.Domain(), team.ChannelName(msgStruct.Channel), team.UserName(msgStruct.Message.Edited.User), team.UserName(msgStruct.Message.User), msgStruct.Message.Text)
				if dumpMessages {
					break typeswitch
				}
				return
			default:
				break typeswitch
			}
			fmt.Printf("[%s%s] [@%s] %s\n", team.Domain(), team.ChannelName(msg.ChannelID()), team.UserName(msg.UserID()), msg.Text())
			if dumpMessages {
				break typeswitch
			}
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
			fmt.Printf("[%s%s] :%s: @%s -> @%s %s\n", team.Domain(), team.ChannelName(channel),
				msg.StringField("reaction"), team.UserName(msg.UserID()),
				team.UserName(slack.UserID(msg.StringField("item_user"))), team.ArchiveURL(slack.MsgID(channel, ts)))
			return
		}
		fmt.Println(colorDebug(fmt.Sprintf("[%s] raw message: %s", team.Domain(), msg)))
	}
}

func readyTeam(cfg *ini.File, name string) (marvin.Team, *rtm.Client, error) {
	teamConfig := marvin.LoadTeamConfig(cfg.Section(name))
	team, err := controller.NewTeam(teamConfig)
	if err != nil {
		return nil, nil, errors.Wrap(err, "NewTeam")
	}
	l, err := net.Listen("tcp4", teamConfig.HTTPListen)
	if err != nil {
		return nil, nil, errors.Wrap(err, "listen tcp")
	}
	client := rtm.NewClient(team)
	client.RegisterRawHandler("main.go", messagePrinter(team, false),
		rtm.MsgTypeAll, nil)

	team.ConnectRTM(client)
	if !team.EnableModules() {
		return nil, nil, errors.Errorf("Some modules failed to load, exiting")
	}
	team.ConnectHTTP(l)

	return team, client, nil
}

func main() {
	teamNamesStr := flag.String("team", "Test", "which team to use")
	configFile := flag.String("conf", "", "override config file")
	// dumpMessages := flag.Bool("msgdump", false, "dump message events")
	flag.Parse()

	var cfg *ini.File
	var err error
	if *configFile != "" {
		cfg, err = ini.Load(*configFile)
	} else {
		cfg, err = ini.LooseLoad("testdata/config.ini", "config.ini")
	}
	if err != nil {
		util.LogError(errors.Wrap(err, "loading config"))
		os.Exit(9)
	}

	teamNames := strings.Split(*teamNamesStr, ",")
	teams := make([]marvin.Team, len(teamNames))
	rtmClients := make([]*rtm.Client, len(teamNames))
	for i, name := range teamNames {
		teams[i], rtmClients[i], err = readyTeam(cfg, name)
		if err != nil {
			util.LogError(err)
			os.Exit(9)
		}
	}

	for _, v := range rtmClients {
		go v.Start()
	}

	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, syscall.SIGINT)
	<-signalCh

	var wg sync.WaitGroup
	wg.Add(len(teams))
	for _, v := range teams {
		go func(t marvin.Team) {
			defer wg.Done()
			t.Shutdown()
		}(v)
	}
	wg.Wait()
	os.Exit(14)
	return
}
