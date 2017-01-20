package intra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/riking/marvin"
	"github.com/riking/marvin/util"
	"golang.org/x/oauth2"
)

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

func OAuthConfig(team marvin.Team) oauth2.Config {
	return oauth2.Config{
		ClientID:     team.TeamConfig().IntraUID,
		ClientSecret: team.TeamConfig().IntraSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://api.intra.42.fr/oauth/authorize",
			TokenURL: "https://api.intra.42.fr/oauth/token",
		},
		RedirectURL: team.AbsoluteURL("/oauth/intra/callback"),
		Scopes:      []string{},
	}
}

type Helper struct {
	*http.Client
	Config oauth2.Config
	Token  *oauth2.Token
}

func Client(ctx context.Context, config oauth2.Config, token *oauth2.Token) *Helper {
	return &Helper{
		Client: config.Client(ctx, token),
		Config: config,
		Token:  token,
	}
}

// GetJSON returns a http.Response with a closed body, the body having been json-unmarshaled into v.
// Method should be something along the lines of "/v2/me".
func (h *Helper) getJSON(ctx context.Context, uri *url.URL, v interface{}) (*http.Response, error) {
	uri.Scheme = "https"
	uri.Host = "api.intra.42.fr"
	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "intra: bad request URI [%s]", uri.RequestURI())
	}
	fmt.Println("intra: doing GET", req.URL.String())
	req = req.WithContext(ctx)
	resp, err := h.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "intra: failed GET %s", uri.RequestURI())
	}
	if resp.StatusCode == 401 {
		h.Token.Expiry = time.Now().Add(-1 * time.Minute)
		// TODO use client credentials instead
		resp, err = h.Do(req)
		if err != nil {
			return nil, errors.Wrapf(err, "intra: failed GET %s", uri.RequestURI())
		}
	}

	err = json.NewDecoder(resp.Body).Decode(v)
	resp.Body.Close()
	if err != nil {
		return nil, errors.Wrapf(err, "intra: failed json decode for GET %s", uri.RequestURI())
	}
	return resp, err
}

func (h *Helper) DoGetFormJSON(ctx context.Context, path string, params url.Values, v interface{}) (*http.Response, error) {
	reqURI, err := methodFormToFinalURL(path, params)
	if err != nil {
		return nil, err
	}
	return h.getJSON(ctx, reqURI, v)
}

type PaginatedSingleResult struct {
	OK    bool
	Error error
	Value interface{}
}

// PaginatedGet loads all pages of Intra's response to the given query.
// Entries in params are overridden by query parameters contained in `method`.
// receiverType should be a pointer to the type of values you want in the result channel, but will not actually be used to store the result.
//
// Any errors encountered will be the last value sent over the returned channel, and the returned channel will always be closed once all results have been sent.
func (h *Helper) PaginatedGet(ctx context.Context, method string, _form url.Values, receiverType interface{}) <-chan PaginatedSingleResult {
	uri, form, err := methodFormToUriForm(method, _form)
	if err != nil {
		ch := make(chan PaginatedSingleResult, 1)
		ch <- PaginatedSingleResult{OK: false, Error: err}
		close(ch)
		return ch
	}

	typ := reflect.TypeOf(receiverType)
	pointedToType := typ.Elem()
	resultCh := make(chan PaginatedSingleResult)
	go func() {
		defer func(ch chan PaginatedSingleResult) {
			close(ch)
		}(resultCh)
		defer func() {
			if rErr := recover(); rErr != nil {
				var qErr error
				if err, ok := rErr.(error); ok {
					qErr = err
				} else {
					qErr = errors.Errorf("Panic: %v", rErr)
				}
				util.LogError(qErr)
				resultCh <- PaginatedSingleResult{OK: false, Error: qErr}
			}
		}()

		for page := 1; ; page++ {
			form.Set("page", strconv.Itoa(page))
			uri.RawQuery = form.Encode()

			aryPtr := reflect.New(reflect.SliceOf(pointedToType))
			resp, err := h.getJSON(ctx, uri, aryPtr.Interface())
			if err != nil {
				resultCh <- PaginatedSingleResult{OK: false, Error: errors.Wrapf(err, "intra GET %s", uri.RequestURI())}
				return
			}

			ary := aryPtr.Elem()
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
			if page*perPage > total {
				// All done!
				return
			}
			select {
			case <-ctx.Done(): // cancelled
				resultCh <- PaginatedSingleResult{Error: ctx.Err(), OK: false}
				return
			default:
				// page++, loop
			}
		}
	}()

	return resultCh
}

const luahelp = `
intra.campus.paris = 1
intra.campus.fremont = 7
intra.project.get_next_line.id
`

type errUrlFormMissing struct{ ParamName string }

func (e errUrlFormMissing) Error() string {
	return fmt.Sprintf("missing mandatory URL parameter: %s", e.ParamName)
}

var rgxUrlParam = regexp.MustCompile(`:([a-zA-Z_]+)`)

func subPathPlaceholders(path string, params url.Values) (string, error) {
	var m = rgxUrlParam.FindStringIndex(path)
	for ; m != nil; m = rgxUrlParam.FindStringIndex(path) {
		paramName := path[m[0]+1 : m[1]]
		if params.Get(paramName) == "" {
			return "", errUrlFormMissing{ParamName: paramName}
		}
		path = path[0:m[0]] + params.Get(paramName) + path[m[1]:]
		params.Del(paramName)
	}
	return path, nil
}

func methodFormToUriForm(method string, form url.Values) (*url.URL, url.Values, error) {
	uri, err := url.Parse(fmt.Sprintf("https://api.intra.42.fr%s", method))
	if err != nil {
		return nil, nil, errors.Wrap(err, "intra: bad request URI")
	}

	uri.Path, err = subPathPlaceholders(uri.Path, form)
	if err != nil {
		return nil, nil, errors.Wrap(err, "intra: bad request URI")
	}
	for k, v := range uri.Query() {
		form[k] = v
	}
	return uri, form, nil
}

func methodFormToFinalURL(method string, form url.Values) (*url.URL, error) {
	uri, f, err := methodFormToUriForm(method, form)
	if err != nil {
		return nil, err
	}
	uri.RawQuery = f.Encode()
	return uri, nil
}
