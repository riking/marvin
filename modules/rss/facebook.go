package rss

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/util"
)

const phpFormat = "2006-01-02T15:04:05-0700"

// Twitter
// https://apps.twitter.com/app/13374175
// Facebook
// https://developers.facebook.com/apps/680115492170112
// ?fields=name,link,feed{description,message,full_picture,created_time,story},username

const facebookFavicon = "https://static.xx.fbcdn.net/rsrc.php/yV/r/hzMapiNYYpW.ico"
const facebookColor = "#3b5998"

type PHPTime struct{ time.Time }

func (t *PHPTime) UnmarshalJSON(data []byte) error {
	var err error
	t.Time, err = time.Parse(`"`+phpFormat+`"`, string(data))
	return err
}

type FacebookFeed struct {
	Name     string
	Username string
	Link     string
	Feed     struct {
		Data []FacebookFeedDataItem
	}
	ID string
}

type FacebookFeedDataItem struct {
	Message     string
	Story       string `json:"story"`
	Description string

	PermalinkURL string  `json:"permalink_url"`
	CreatedTime  PHPTime `json:"created_time"`
	FullPicture  string  `json:"full_picture"`
	ID           string  `json:"id"`
}

func (f FacebookFeedDataItem) Render(parent *FacebookFeed) slack.OutgoingSlackMessage {
	var buf bytes.Buffer
	var msg slack.OutgoingSlackMessage

	msg.UnfurlLinks = util.TriNo
	atch := slack.Attachment{}
	atch.Color = facebookColor
	atch.Fallback = f.PermalinkURL
	atch.AuthorIcon = facebookFavicon
	atch.AuthorName = parent.Name
	atch.AuthorLink = parent.Link
	if f.Story != "" {
		buf.WriteString(f.Story)
	} else {
		fmt.Fprintf(&buf, "%s made a new post on Facebook.", parent.Name)
	}
	atch.Title = fmt.Sprintf("New post on %s", parent.Username)
	atch.TitleLink = f.PermalinkURL
	if f.Message != "" {
		atch.Text = f.Message
	}
	if f.FullPicture != "" {
		if strings.Contains(f.FullPicture, "safe_image.php") {
			atch.ImageURL = fb_safeImageExtract(f.FullPicture)
		}
		atch.ImageURL = f.FullPicture
	}

	if f.Description != "" {
		// Split into two attachments
		msg.Attachments = append(msg.Attachments, atch)
		atch = slack.Attachment{}
		atch.Color = facebookColor
		atch.Text = f.Description
	}
	atch.Footer = "Use @marvin rss to manage"
	atch.TS = f.CreatedTime.Unix()

	msg.Attachments = append(msg.Attachments, atch)
	msg.Text = buf.String()
	return msg
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
