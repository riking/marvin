package intra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"net/url"
	"reflect"
	"strconv"
	"github.com/pkg/errors"
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

type PaginatedSingleResult struct {
	OK bool
	Error error
	Value interface{}
}

// PaginatedGet loads all pages of Intra's response to the given query.
// Entries in params are overridden by query parameters contained in `method`.
// receiverType should be a pointer to the type of values you want in the result channel, but will not actually be used to store the result.
func (h *Helper) PaginatedGet(method, path string, params url.Values, receiverType interface{}) (chan <-PaginatedSingleResult) {
	uri, err := url.Parse(fmt.Sprintf("https://api.intra.42.fr%s", path))
	if err != nil {
		panic(err)
	}
	for k, v := range uri.Query() {
		params[k] = v
	}

	typ := reflect.TypeOf(receiverType)
	pointedToType := typ.Elem()
	resultCh := make(chan PaginatedSingleResult)
	go func() {
		defer func(ch chan PaginatedSingleResult) {
			close(ch)
		}(resultCh)

		for page := 1; ; page++ {
			params.Set("page", strconv.Itoa(page))
			uri.RawQuery = params.Encode()

			ary := reflect.MakeSlice(pointedToType, 0, 20)
			resp, err := h.GetJSON(uri.RequestURI(), ary.Addr().Interface())
			if err != nil {
				resultCh <- PaginatedSingleResult{OK: false, Error: errors.Wrapf(err, "intra GET %s", uri.RequestURI())}
				return
			}

			l := ary.Len()
			for i := 0; i < l; i++ {
				resultCh <- PaginatedSingleResult{
					OK:    true,
					Value: ary.Index(i).Addr().Interface(),
				}
				continue
			}

			h := resp.Header
			perPage, err := strconv.Atoi(h.Get("X-Per-Page"))
			if err != nil {
				resultCh <- PaginatedSingleResult{OK: false, Error: errors.Errorf("intra GET %s: X-Per-Page not a number", uri.RequestURI())}
				return
			}
			total, err := strconv.Atoi(h.Get("X-Total"))
			if err != nil {
				resultCh <- PaginatedSingleResult{OK: false, Error: errors.Errorf("intra GET %s: X-Per-Page not a number", uri.RequestURI())}
				return
			}
			if page * perPage > total {
				// All done!
				return
			}
			// page++, loop
		}
	}()

	return resultCh
}

const luahelp = `
intra.campus.paris = 1
intra.campus.fremont = 7
intra.project.get_next_line.id
`

func (h *Helper) GetProjectRegisteredUsers(projectName string) {

}
