package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/pkg/errors"
	"github.com/riking/marvin"
	"github.com/riking/marvin/intra"
	"github.com/riking/marvin/modules/weblogin"
	"github.com/riking/marvin/slack/controller"
	"github.com/riking/marvin/util"
	"gopkg.in/ini.v1"
)

func main() {
	configFile := flag.String("conf", "", "override config file")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: intra-cli /v2/request-endpoint")
		os.Exit(1)
	}

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
	team.EnableModules()
	team.EnableModule(weblogin.Identifier)

	ctx := context.Background()

	user, err := team.GetModule(weblogin.Identifier).(weblogin.API).GetUserByIntra("kyork")
	if err != nil || user.IntraToken == nil {
		util.LogError(errors.Wrap(err, "getting user"))
		return
	}

	client := intra.Client(ctx, intra.OAuthConfig(team), user.IntraToken)
	kyorkID, err := client.UserIDByLogin(ctx, "kyork")
	if err != nil {
		util.LogError(errors.Wrap(err, "get kyork user id"))
		return
	}

	var result interface{}

	_, err = client.DoGetFormJSON(ctx, flag.Arg(0), url.Values{}, &userVal)
	if err != nil {
		util.LogError(errors.Wrap(err, "fetch"))
		return
	}
	b, err := json.MarshalIndent(userVal, "", "\t")
	if err != nil {
		util.LogError(err)
	}
	fmt.Println(string(b))
}
