package main

import (
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/ini.v1"

	"github.com/riking/homeapi/shocky"
	"github.com/riking/homeapi/shocky/slack"
	"github.com/riking/homeapi/shocky/slack/controller"
	"github.com/riking/homeapi/shocky/slack/rtm"

	_ "github.com/riking/homeapi/shocky/modules/at_command"
	_ "github.com/riking/homeapi/shocky/modules/autoinvite"
)

func main() {
	cfg, err := ini.LooseLoad("testdata/config.ini", "config.ini", "/tank/www/apiserver/config.ini")
	if err != nil {
		log.Fatalln(errors.Wrap(err, "loading config"))
	}
	//mainSect := cfg.Section("Shocky")
	//teamListK, err := mainSect.GetKey("Teams")
	//if err != nil {
	//	log.Fatalln(errors.Wrap(err, "no key Shocky.Teams"))
	//}
	//teamList := teamListK.Strings(",")
	teamConfig := shocky.LoadTeamConfig(cfg.Section("ShockyTest"))
	team := controller.NewTeam(teamConfig)
	client, err := rtm.Dial(team)
	if err != nil {
		log.Fatalln(errors.Wrap(err, "rtm.Dial"))
	}
	client.RegisterRawHandler("debug", func(msg slack.RTMRawMessage) {
		fmt.Println("[DEBUG]", "main.go rtm message:", msg)
	}, rtm.MsgTypeAll, nil)
	team.Connect(client)
	client.Start()

	fmt.Println("started")
	time.Sleep(3*time.Second)
	time.Sleep(60 * time.Second)
}
