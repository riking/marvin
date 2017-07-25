package intra

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

func ClientCredentialsTokenSource(ctx context.Context, uid, secret string, scopes ...string) oauth2.TokenSource {
	c := clientcredentials.Config{
		ClientID:     uid,
		ClientSecret: secret,
		TokenURL:     "https://api.intra.42.fr/oauth/token",
		Scopes:       []string{"projects", "", "public"},
	}
	return c.TokenSource(ctx)
}

type Helper struct {
	*http.Client
}

func Client(ctx context.Context, toksource oauth2.TokenSource) *Helper {
	return &Helper{
		Client: oauth2.NewClient(ctx, toksource),
	}
}

// GetJSON returns a http.Response with a closed body, the body having been json-unmarshaled into v.
// Method should be something along the lines of "/v2/me".
func (h *Helper) getJSON(ctx context.Context, uri *url.URL, httpMethod string, v interface{}) (*http.Response, error) {
	uri.Scheme = "https"
	uri.Host = "api.intra.42.fr"
	req, err := http.NewRequest(httpMethod, uri.String(), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "intra: bad request URI [%s]", uri.RequestURI())
	}
	fmt.Fprintln(os.Stderr, "intra: doing GET", req.URL.String())
	req = req.WithContext(ctx)
	resp, err := h.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "intra: failed GET %s", uri.RequestURI())
	}
	if resp.StatusCode == 401 {
		resp, err = h.Do(req)
		if err != nil {
			return nil, errors.Wrapf(err, "intra: failed GET %s", uri.RequestURI())
		}
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, errors.Wrapf(err, "intra: failed read for GET %s", uri.RequestURI())
	}
	err = json.Unmarshal(bytes, v)
	resp.Body.Close()
	if err != nil {
		return nil, errors.Wrapf(err, "intra: failed json decode for GET %s, %s", uri.RequestURI(), string(bytes))
	}
	return resp, err
}

func (h *Helper) DoGetFormJSON(ctx context.Context, path string, params url.Values, v interface{}) (*http.Response, error) {
	reqURI, httpMethod, err := methodFormToFinalURL(path, params)
	if err != nil {
		return nil, err
	}
	return h.getJSON(ctx, reqURI, httpMethod, v)
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
	httpMethod := "GET"
	if form.Get("_method") != "" {
		httpMethod = form.Get("_method")
		form.Del("_method")
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
				fmt.Fprintln(os.Stderr, qErr)
				resultCh <- PaginatedSingleResult{OK: false, Error: qErr}
			}
		}()

		for page := 1; ; page++ {
			form.Set("page", strconv.Itoa(page))
			uri.RawQuery = form.Encode()

			aryPtr := reflect.New(reflect.SliceOf(pointedToType))
			resp, err := h.getJSON(ctx, uri, httpMethod, aryPtr.Interface())
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

// PaginatedGet loads all pages of Intra's response to the given query.
// Entries in params are overridden by query parameters contained in `method`.
// receiverType should be a pointer to the type of values you want in the result channel, but will not actually be used to store the result.
//
// Any errors encountered will be the last value sent over the returned channel, and the returned channel will always be closed once all results have been sent.
//
// This works for endpoints that return data as {"data": [...], "links": {...}} .
func (h *Helper) PaginatedGetLinkStyle(ctx context.Context, method string, _form url.Values, receiverType interface{}) <-chan PaginatedSingleResult {
	uri, form, err := methodFormToUriForm(method, _form)
	if err != nil {
		ch := make(chan PaginatedSingleResult, 1)
		ch <- PaginatedSingleResult{OK: false, Error: err}
		close(ch)
		return ch
	}
	httpMethod := "GET"
	if form.Get("_method") != "" {
		httpMethod = form.Get("_method")
		form.Del("_method")
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
				fmt.Fprintln(os.Stderr, qErr)
				resultCh <- PaginatedSingleResult{OK: false, Error: qErr}
			}
		}()

		var decode struct {
			Data  json.RawMessage
			Links struct {
				Self string
				Next string
				Last string
			}
		}

		for page := 1; ; page++ {
			form.Set("page", strconv.Itoa(page))
			uri.RawQuery = form.Encode()

			resp, err := h.getJSON(ctx, uri, httpMethod, &decode)
			if err != nil {
				resultCh <- PaginatedSingleResult{OK: false, Error: errors.Wrapf(err, "intra GET %s", uri.RequestURI())}
				return
			}

			aryPtr := reflect.New(reflect.SliceOf(pointedToType))
			err = json.Unmarshal([]byte(decode.Data), aryPtr.Interface())
			if err != nil {
				resultCh <- PaginatedSingleResult{OK: false, Error: errors.Wrapf(err, "json unmarshal %s", string(decode.Data))}
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

func methodFormToFinalURL(method string, form url.Values) (*url.URL, string, error) {
	uri, f, err := methodFormToUriForm(method, form)
	if err != nil {
		return nil, "", err
	}
	httpMethod := "GET"
	if form.Get("_method") != "" {
		httpMethod = form.Get("_method")
		form.Del("_method")
	}
	uri.RawQuery = f.Encode()
	return uri, httpMethod, nil
}
