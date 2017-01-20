package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
	"github.com/riking/marvin"
	"github.com/riking/marvin/intra"
	"github.com/riking/marvin/modules/weblogin"
	"github.com/riking/marvin/slack/controller"
	"github.com/riking/marvin/util"
	"gopkg.in/ini.v1"
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

	var userVal map[string]interface{}
	_, err = client.DoGetFormJSON(ctx, "/v2/users/:id", url.Values{"id": []string{strconv.Itoa(kyorkID)}}, &userVal)
	if err != nil {
		util.LogError(errors.Wrap(err, "get kyork user"))
		return
	}
	b, err := json.MarshalIndent(userVal, "", "\t")
	if err != nil {
		util.LogError(err)
	}
	fmt.Println(string(b))

	return

	form := url.Values{}
	form.Set("user_id", strconv.Itoa(kyorkID))
	form.Set("filter[campus]", "7")
	ch := client.PaginatedGet(ctx, "/v2/users/:user_id/projects_users", form, new(intra.ProjectUser))
	for resp := range ch {
		if !resp.OK {
			util.LogError(resp.Error)
			break
		}
		fmt.Printf("type: %T", resp.Value)
		b, err := json.MarshalIndent(resp.Value, "", "\t")
		if err != nil {
			util.LogError(err)
		}
		fmt.Println(string(b))
	}
}
