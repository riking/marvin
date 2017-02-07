package rss

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/riking/marvin"
	"github.com/riking/marvin/modules/rss/twitter"
	"github.com/riking/marvin/slack"
)

// Twitter
// https://apps.twitter.com/app/13374175
//
// statuses/user_timeline

const twitterFavicon = "https://abs.twimg.com/favicons/favicon.ico"
const twitterColor = "#1da1f2"
const twitterAPIRoot = "https://api.twitter.com/1.1"
const twitterTokenURL = twitterAPIRoot + "/oauth2/token"

type TwitterError struct {
	ResponseCode int `json:"-"`
	Errors       []struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"errors"`
}

func (e TwitterError) Error() string {
	if len(e.Errors) == 0 {
		return fmt.Sprintf("Twitter: HTTP response code %d", e.ResponseCode)
	} else {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "Twitter %d:", e.ResponseCode)
		for _, v := range e.Errors {
			fmt.Fprintf(&buf, " %s (%d)", v.Message, v.Code)
		}
		return buf.String()
	}
}

type TwitterFeedDataItem struct {
	Tweet *twitter.Tweet
}

type TwitterFeed struct {
	feedID    string
	FeedLogin string
}

type TwitterType struct {
	mod *RSSModule

	clLock sync.Mutex
	client marvin.HTTPDoer
}

func (t *TwitterType) TypeID() TypeID { return feedTypeTwitter }
func (t *TwitterType) Name() string   { return "twitter" }

func (t *TwitterType) OnLoad(mod *RSSModule) {
	mod.Config().AddProtect("twitter-clientid", "", true)
	mod.Config().AddProtect("twitter-clientsecret", "", true)
	mod.Config().OnModify(func(key string) {
		if strings.HasPrefix(key, "twitter-") {
			// key changed, invalidate cache
			t.clLock.Lock()
			t.client = nil
			t.clLock.Unlock()
		}
	})
	t.mod = mod
}

func (t *TwitterType) OAuthConfig() (clientcredentials.Config, error) {
	clientID, err := t.mod.Config().Get("twitter-clientid")
	if clientID == "" || err != nil {
		return clientcredentials.Config{}, ErrNotConfigured
	}
	clientSecret, err := t.mod.Config().Get("twitter-clientsecret")
	if clientSecret == "" || err != nil {
		return clientcredentials.Config{}, ErrNotConfigured
	}
	return clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     twitterTokenURL,
	}, nil
}

func (t *TwitterType) Client() (marvin.HTTPDoer, error) {
	t.clLock.Lock()
	defer t.clLock.Unlock()
	if t.client == nil {
		if t.client != nil {
			return t.client, nil
		}
		config, err := t.OAuthConfig()
		if err != nil {
			return nil, err
		}
		t.client = config.Client(context.Background())
		return t.client, nil
	}
	cl := t.client
	return cl, nil
}

func (t *TwitterType) Domains() []string {
	return []string{"twitter.com", "www.twitter.com"}
}

const (
	twOptNoRetweets  = "no_retweets"
	twOptWithReplies = "with_replies"
)

func (t *TwitterType) VerifyFeedIdentifier(ctx context.Context, input string) (string, error) {
	var screenName string

	uri, err := url.Parse(input)
	if err != nil {
		return "", errors.Wrap(err, "input must be in the form of a URI")
	}
	screenName = uri.Path
	if strings.HasPrefix(screenName, "/") {
		screenName = screenName[1:]
	}
	if strings.HasPrefix(screenName, "@") {
		screenName = screenName[1:]
	}
	opts := uri.Query()
	noRTs := false
	withReplies := false
	if strings.HasSuffix(screenName, "/with_replies") {
		screenName = strings.TrimSuffix(screenName, "/with_replies")
		withReplies = true
	}
	if _, present := opts[twOptNoRetweets]; present {
		noRTs = true
	}
	if _, present := opts[twOptWithReplies]; present {
		withReplies = true
	}

	client, err := t.Client()
	if err != nil {
		return "", err
	}
	uri, err = url.Parse(twitterAPIRoot + "/users/show.json")
	if err != nil {
		return "", err
	}
	form := url.Values{
		"screen_name":      []string{screenName},
		"include_entities": []string{"false"},
	}
	uri.RawQuery = form.Encode()
	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return "", err
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "Twitter: Error checking @%s", screenName)
	}
	var response struct {
		TwitterError
		ScreenName string `json:"screen_name"`
	}
	response.TwitterError.ResponseCode = resp.StatusCode
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", errors.Wrap(err, "Twitter API json decode error")
	}
	if response.TwitterError.Errors != nil {
		return "", errors.Wrapf(response.TwitterError, "Checking for @%s", screenName)
	}

	opts = url.Values{}
	if noRTs {
		opts[twOptNoRetweets] = []string{"1"}
	}
	if withReplies {
		opts[twOptWithReplies] = []string{"1"}
	}
	return response.ScreenName + "?" + opts.Encode(), nil
}

func (t *TwitterType) LoadFeed(ctx context.Context, feedID string, lastSeen string) (FeedMeta, []Item, error) {
	uri, err := url.Parse(feedID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Bad Twitter FeedID '%s'", feedID)
	}
	screenName := uri.Path
	opts := uri.Query()
	noRTs := false
	withReplies := false
	if _, present := opts[twOptNoRetweets]; present {
		noRTs = true
	}
	if _, present := opts[twOptWithReplies]; present {
		withReplies = true
	}

	client, err := t.Client()
	if err != nil {
		return nil, nil, err
	}
	uri, err = url.Parse(twitterAPIRoot + "/statuses/user_timeline.json")
	if err != nil {
		return nil, nil, err
	}
	form := url.Values{
		"screen_name": []string{screenName},
	}
	if lastSeen != "" {
		form.Set("since_id", lastSeen)
	}
	if !withReplies {
		form.Set("exclude_replies", "true")
	}
	if noRTs {
		form.Set("include_rts", "false")
	}
	uri.RawQuery = form.Encode()
	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	type normalResponse []*twitter.Tweet
	var response struct {
		TwitterError
		normalResponse
	}
	response.TwitterError.ResponseCode = resp.StatusCode
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, nil, err
	}
	if response.TwitterError.Errors != nil {
		return nil, nil, response.TwitterError
	}
	meta := &TwitterFeed{
		feedID:    feedID,
		FeedLogin: screenName,
	}
	itemSlice := make([]Item, len(response.normalResponse))
	for i, v := range response.normalResponse {
		itemSlice[i] = TwitterFeedDataItem{Tweet: v}
	}
	return meta, itemSlice, nil
}

func (f *TwitterFeed) FeedID() string { return f.feedID }

func (i TwitterFeedDataItem) Render(p FeedMeta) slack.OutgoingSlackMessage {
	var buf bytes.Buffer
	var msg slack.OutgoingSlackMessage
	tf := p.(*TwitterFeed)

	fmt.Fprintf(&buf, "New tweet by <https://twitter.com/%s|@%s>\n", tf.FeedLogin, tf.FeedLogin)
	fmt.Fprintf(&buf, "https://twitter.com/%s/%s", i.Tweet.Source, i.Tweet.IDStr)
	msg.Text = buf.String()
	return msg
}

func (i TwitterFeedDataItem) ItemID() string {
	return i.Tweet.IDStr
}
