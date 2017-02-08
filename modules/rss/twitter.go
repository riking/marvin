package rss

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"

	"github.com/riking/marvin"
	"github.com/riking/marvin/modules/rss/twitter"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/util"
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
	mod.Config().AddProtect("twitter-token", "", true)
	mod.Config().AddProtect("twitter-tokensecret", "", true)
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

func (t *TwitterType) Client() (marvin.HTTPDoer, error) {
	t.clLock.Lock()
	defer t.clLock.Unlock()
	if t.client == nil {
		conf := t.mod.Config()
		clientID, err := conf.Get("twitter-clientid")
		if clientID == "" || err != nil {
			return nil, ErrNotConfigured
		}
		clientSecret, err := conf.Get("twitter-clientsecret")
		if clientSecret == "" || err != nil {
			return nil, ErrNotConfigured
		}
		tokStr, err := conf.Get("twitter-token")
		if tokStr == "" || err != nil {
			return nil, ErrNotConfigured
		}
		tokSec, err := conf.Get("twitter-tokensecret")
		if tokStr == "" || err != nil {
			return nil, ErrNotConfigured
		}
		config := oauth1.NewConfig(clientID, clientSecret)
		token := oauth1.NewToken(tokStr, tokSec)
		t.client = config.Client(context.Background(), token)
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
		"tweet_mode":  []string{"extended"},
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
	var normalResponse []*twitter.Tweet
	var errResponse TwitterError
	errResponse.ResponseCode = resp.StatusCode
	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, nil, err
	}
	_ = json.Unmarshal(b, &errResponse)
	if errResponse.Errors != nil {
		return nil, nil, errResponse
	}
	err = json.Unmarshal(b, &normalResponse)
	if err != nil {
		return nil, nil, err
	}
	meta := &TwitterFeed{
		feedID:    feedID,
		FeedLogin: screenName,
	}
	itemSlice := make([]Item, len(normalResponse))
	for i, v := range normalResponse {
		itemSlice[i] = TwitterFeedDataItem{Tweet: v}
	}
	return meta, itemSlice, nil
}

func (f *TwitterFeed) FeedID() string { return f.feedID }

func (i TwitterFeedDataItem) Render(p FeedMeta) slack.OutgoingSlackMessage {
	var buf bytes.Buffer
	var msg slack.OutgoingSlackMessage
	tf := p.(*TwitterFeed)

	fmt.Fprintf(&buf, "New tweet by <https://twitter.com/%s|@%s>: <https://twitter.com/%s/%s>",
		tf.FeedLogin, tf.FeedLogin, i.Tweet.User.ScreenName, i.Tweet.IDStr)
	msg.Text = buf.String()
	msg.Parse = "none"
	msg.UnfurlLinks = util.TriNo
	msg.UnfurlMedia = util.TriNo

	tweet := i.Tweet
	if tweet.RetweetedStatus != nil {
		tweet = tweet.RetweetedStatus
	}
	var atch slack.Attachment
	atch.Color = twitterColor
	atch.Fallback = tweet.Text
	ts, _ := time.Parse(time.RubyDate, tweet.CreatedAt)
	atch.TS = ts.Unix()
	atch.AuthorIcon = twitterFavicon
	atch.AuthorName = tweet.User.Name
	atch.AuthorLink = fmt.Sprintf("https://twitter.com/%s", tweet.User.ScreenName)
	atch.AuthorIcon = tweet.User.ProfileImageURLHttps
	atch.AuthorSubname = "@" + tweet.User.ScreenName
	atch.Footer = "Twitter"
	atch.FooterIcon = "https://a.slack-edge.com/6e067/img/services/twitter_pixel_snapped_32.png"
	atch.Text = tweetToSlackText(tweet)
	atch.FromURL = fmt.Sprintf("https://twitter.com/%s/%s", tweet.User.ScreenName, tweet.IDStr)
	atch.ServiceName = "twitter"
	atch.ServiceURL = "https://twitter.com/"

	if tweet.ExtendedEntities != nil && len(tweet.ExtendedEntities.Media) == 1 {
		atch.ImageURL = tweet.Entities.Media[0].MediaURLHttps
	} else if tweet.ExtendedEntities != nil && len(tweet.ExtendedEntities.Media) > 1 {
		atch.ImageURL = tweet.Entities.Media[0].MediaURLHttps
		msg.Attachments = append(msg.Attachments, atch)
		atch = slack.Attachment{}
		atch.Color = twitterColor
		atch.Text = fmt.Sprintf("<https://twitter.com/%s/%s|%d more photos not shown>",
			tweet.User.ScreenName, tweet.IDStr, len(tweet.ExtendedEntities.Media)-1)
	} else if len(tweet.Entities.Media) == 1 {
		atch.ImageURL = tweet.Entities.Media[0].MediaURLHttps
	}

	msg.Attachments = append(msg.Attachments, atch)
	return msg
}

func (i TwitterFeedDataItem) ItemID() string {
	return i.Tweet.IDStr
}

type twitterEntityReplacement struct {
	twitter.Indices
	Replacement string
}

func tweetToSlackText(t *twitter.Tweet) string {
	var replaces []twitterEntityReplacement
	for _, v := range t.Entities.Hashtags {
		replaces = append(replaces, twitterEntityReplacement{
			Indices: v.Indices,
			Replacement: fmt.Sprintf(
				"<https://twitter.com/hashtag/%s?src=hash|#%s>", url.PathEscape(v.Text), v.Text),
		})
	}
	for _, v := range t.Entities.Symbols {
		replaces = append(replaces, twitterEntityReplacement{
			Indices: v.Indices,
			Replacement: fmt.Sprintf(
				"<https://twitter.com/hashtag/%s?src=hash|$%s>", url.PathEscape(v.Text), v.Text),
		})
	}
	for _, v := range t.Entities.UserMentions {
		replaces = append(replaces, twitterEntityReplacement{
			Indices: v.Indices,
			Replacement: fmt.Sprintf(
				"<https://twitter.com/%s|@%s>", v.ScreenName, v.ScreenName),
		})
	}
	for _, v := range t.Entities.Urls {
		replaces = append(replaces, twitterEntityReplacement{
			Indices:     v.Indices,
			Replacement: fmt.Sprintf("<%s|%s>", v.ExpandedURL, v.DisplayURL),
		})
	}
	for _, v := range t.Entities.Media {
		replaces = append(replaces, twitterEntityReplacement{
			Indices:     v.Indices,
			Replacement: "",
		})
	}

	sort.Slice(replaces, func(i, j int) bool {
		return replaces[i].Indices[0] < replaces[j].Indices[0]
	})
	var buf bytes.Buffer
	entIdx := 0
	textAsRunes := []rune(t.Text)
	for i := 0; i < len(textAsRunes); {
		if entIdx < len(replaces) {
			buf.WriteString(string(textAsRunes[i:replaces[entIdx].Indices[0]]))
			buf.WriteString(replaces[entIdx].Replacement)
			i = replaces[entIdx].Indices[1]
			entIdx++
		} else {
			buf.WriteString(string(textAsRunes[i:len(t.Text)]))
			i = len(t.Text)
		}
	}
	return buf.String()
}
