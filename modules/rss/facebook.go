package rss

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/util"
)

const phpFormat = "2006-01-02T15:04:05-0700"

// Facebook
// https://developers.facebook.com/apps/680115492170112

const facebookFavicon = "https://static.xx.fbcdn.net/rsrc.php/yV/r/hzMapiNYYpW.ico"
const facebookColor = "#3b5998"
const facebookAPIRoot = "https://graph.facebook.com"
const facebookTokenURL = twitterAPIRoot + "/oauth/access_token"

type PHPTime struct{ time.Time }

func (t *PHPTime) UnmarshalJSON(data []byte) error {
	var err error
	t.Time, err = time.Parse(`"`+phpFormat+`"`, string(data))
	return err
}

type FacebookError struct {
	Message   string `json:"message"`
	Type      string `json:"type"`
	Code      int    `json:"code"`
	FBTraceID string `json:"fbtrace_id"`
}

type FacebookFeed struct {
	Name string
	Link string
	Feed struct {
		Data []FacebookFeedDataItem
	}
	ID string
}

type FacebookType struct {
	mod *RSSModule

	clLock sync.Mutex
	client marvin.HTTPDoer
}

func (t *FacebookType) TypeID() TypeID { return feedTypeFacebook }
func (t *FacebookType) Name() string   { return "facebook" }

func (t *FacebookType) OnLoad(mod *RSSModule) {
	mod.Config().AddProtect("facebook-clientid", "", false)
	mod.Config().AddProtect("facebook-clientsecret", "", true)
	mod.Config().OnModify(func(key string) {
		if strings.HasPrefix(key, "facebook-") {
			// key changed, invalidate cache
			t.clLock.Lock()
			t.client = nil
			t.clLock.Unlock()
		}
	})
	t.mod = mod
}

func (t *FacebookType) OAuthConfig() (clientcredentials.Config, error) {
	clientID, err := t.mod.Config().Get("facebook-clientid")
	if clientID == "" || err != nil {
		return clientcredentials.Config{}, ErrNotConfigured
	}
	clientSecret, err := t.mod.Config().Get("facebook-clientsecret")
	if clientSecret == "" || err != nil {
		return clientcredentials.Config{}, ErrNotConfigured
	}
	return clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     facebookTokenURL,
	}, nil
}

func (t *FacebookType) Client() (marvin.HTTPDoer, error) {
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

func (t *FacebookType) Domains() []string {
	return []string{"facebook.com", "www.facebook.com"}
}

var rgxFacebookPage = regexp.MustCompile(`https://www\.facebook\.com/(?:\w*#!/)?(?:pages/)?([\w\-_]*)(-\d+)?$`)

func (t *FacebookType) VerifyFeedIdentifier(ctx context.Context, input string) (string, error) {
	client, err := t.Client()
	if err != nil {
		return "", err
	}

	var idCandidate string
	m := rgxFacebookPage.FindStringSubmatch(input)
	if m != nil && m[2] != "" {
		idCandidate = m[2]
	} else if m != nil {
		idCandidate = m[1]
	} else {
		idCandidate = input
	}

	req, err := http.NewRequest("GET", fmt.Sprintf(
		"%s/v2.8/%s?fields=id,feed.limit(1){id}", facebookAPIRoot,
		url.PathEscape(idCandidate)), nil)
	if err != nil {
		panic(errors.Wrap(err, "could not construct facebook check url"))
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "Could not contact Facebook API")
	}
	if resp.StatusCode != 200 {
		return "", errors.Errorf("Provided name '%s' does not appear to be a Facebook user/page (response code %d)", idCandidate, resp.StatusCode)
	}
	var response struct {
		FacebookError
		ID string `json:"id"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	resp.Body.Close()
	if err != nil {
		return "", errors.Wrap(err, "Could not decode Facebook API response")
	} else if response.FacebookError.Code != 0 {
		return "", errors.Errorf("Facebook API Error (is this a page?): %s", response.FacebookError.Message)
	}
	return response.ID, nil
}

type FacebookFeedDataItem struct {
	Message     string `json:"message"`
	Story       string `json:"story"`
	Description string `json:"description"`

	PermalinkURL string  `json:"permalink_url"`
	CreatedTime  PHPTime `json:"created_time"`
	FullPicture  string  `json:"full_picture"`
	From         struct {
		Name string `json:"name"`
	} `json:"from"`
	ID string `json:"id"`
}

func (t *FacebookType) LoadFeed(ctx context.Context, feedID string, lastSeen string) (FeedMeta, []Item, error) {
	client, err := t.Client()
	if err != nil {
		return nil, nil, err
	}

	_ = lastSeen

	req, err := http.NewRequest("GET", fmt.Sprintf(
		"%s/v2.8/%s?fields=name,link,feed{message,story,description,permalink_url,full_picture,created_time,from{name}}", facebookAPIRoot,
		feedID), nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to construct URL")
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Could not contact Facebook API")
	}
	var response struct {
		FacebookError
		FacebookFeed
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	resp.Body.Close()
	if err != nil {
		return nil, nil, errors.Wrap(err, "Could not decode Facebook API response")
	} else if response.FacebookError.Code != 0 {
		return nil, nil, errors.Errorf("Facebook API Error: %s", response.FacebookError.Message)
	}

	if response.FacebookFeed.ID == "" {
		return nil, nil, errors.Errorf("Facebook API Error: ID field empty for some reason? %v", response)
	}
	itemSlice := make([]Item, len(response.Feed.Data))
	for i, v := range response.Feed.Data {
		itemSlice[i] = v
	}
	return &response.FacebookFeed, itemSlice, nil
}

func (f *FacebookFeed) FeedID() string { return f.ID }

func (i FacebookFeedDataItem) Render(p FeedMeta) slack.OutgoingSlackMessage {
	var buf bytes.Buffer
	var msg slack.OutgoingSlackMessage

	parent := p.(*FacebookFeed)
	msg.UnfurlLinks = util.TriNo
	atch := slack.Attachment{}
	atch.Color = facebookColor
	atch.Fallback = i.PermalinkURL
	atch.AuthorIcon = facebookFavicon
	atch.AuthorName = i.From.Name
	atch.AuthorLink = parent.Link
	if i.Story != "" {
		buf.WriteString(i.Story)
	} else {
		fmt.Fprintf(&buf, "%s made a new post on Facebook.", parent.Name)
	}
	atch.Title = fmt.Sprintf("New post on %s", parent.Name)
	atch.TitleLink = i.PermalinkURL
	if i.Message != "" {
		atch.Text = i.Message
	}
	if i.FullPicture != "" {
		if strings.Contains(i.FullPicture, "safe_image.php") {
			atch.ImageURL = fb_safeImageExtract(i.FullPicture)
		}
		atch.ImageURL = i.FullPicture
	}

	if i.Description != "" {
		// Split into two attachments
		msg.Attachments = append(msg.Attachments, atch)
		atch = slack.Attachment{}
		atch.Color = facebookColor
		atch.Text = i.Description
	}
	atch.Footer = "Use @marvin rss to manage"
	atch.TS = i.CreatedTime.Unix()

	msg.Attachments = append(msg.Attachments, atch)
	msg.Text = buf.String()
	return msg
}

func (i FacebookFeedDataItem) ItemID() string {
	return i.ID
}

// Since Slack already does safe-images, we don't need to reproxy through Facebook's.
// Extract the original URL from a safe_image.php URL.
func fb_safeImageExtract(safeImageURL string) string {
	u, err := url.Parse(safeImageURL)
	if err != nil {
		return safeImageURL
	}
	if !strings.Contains(u.Path, "safe_image.php") {
		return safeImageURL
	}
	if u.Host != "external.xx.fbcdn.net" {
		return safeImageURL
	}
	childUrl := u.Query().Get("url")
	if childUrl == "" {
		return safeImageURL
	}
	return childUrl
}
