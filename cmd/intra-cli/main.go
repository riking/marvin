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

	teamConfig := marvin.LoadTeamConfig(cfg.Section("Test"))
	team, err := controller.NewTeam(teamConfig)
	if err != nil {
		util.LogError(errors.Wrap(err, "NewTeam"))
		return
	}
	team.EnableModules()
	team.EnableModule(weblogin.Identifier)

	ctx := context.Background()

	client := intra.Client(ctx, intra.ClientCredentialsTokenSource(ctx, team.TeamConfig().IntraUID, team.TeamConfig().IntraSecret))

	var result interface{}

	_, err = client.DoGetFormJSON(ctx, flag.Arg(0), url.Values{}, &result)
	if err != nil {
		util.LogError(errors.Wrap(err, "fetch"))
		return
	}
	b, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		util.LogError(err)
	}
	fmt.Println(string(b))
}
