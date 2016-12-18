package intra

import (
	"net/http"
	"golang.org/x/oauth2"
	"context"
	"fmt"
	"encoding/json"
)

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type Helper struct {
	*http.Client
	Config oauth2.Config
	Token  *oauth2.Token
}

func Client(config oauth2.Config, token *oauth2.Token) Helper {
	return Helper{
		Client: config.Client(context.Background(), token),
		Config: config,
		Token:  token,
	}
}

// GetJSON returns a http.Response with a closed body, the body having been json-unmarshaled into v.
// Method should be something along the lines of "/v2/me".
func (h *Helper) GetJSON(method string, v interface{}) (*http.Response, error) {
	resp, err := h.Get(fmt.Sprintf("https://api.intra.42.fr%s", method))
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(resp.Body).Decode(v)
	return resp, err
}
