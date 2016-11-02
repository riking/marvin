package slack

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin/util"
)

func SlackAPILog(resp *http.Response, err error) {
	if err != nil {
		util.LogError(err)
	}
	var response struct {
		*APIResponse
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		util.LogError(errors.Wrap(err, "decode json"))
	}
	if !response.OK {
		util.LogError(errors.Wrap(response, "Slack error"))
	}
}
