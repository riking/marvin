package main

import (
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/ini.v1"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/slack/controller"
	"github.com/riking/homeapi/marvin/slack/rtm"

	_ "github.com/riking/homeapi/marvin/modules/at_command"
	_ "github.com/riking/homeapi/marvin/modules/autoinvite"
)

func main() {
	cfg, err := ini.LooseLoad("testdata/config.ini", "config.ini", "/tank/www/apiserver/config.ini")
	if err != nil {
		log.Fatalln(errors.Wrap(err, "loading config"))
	}
	//mainSect := cfg.Section("marvin")
	//teamListK, err := mainSect.GetKey("Teams")
	//if err != nil {
	//	log.Fatalln(errors.Wrap(err, "no key marvin.Teams"))
	//}
	//teamList := teamListK.Strings(",")
	teamConfig := marvin.LoadTeamConfig(cfg.Section("ShockyTest"))
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
