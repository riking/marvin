package githook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/riking/marvin/modules/atcommand"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/util"
)

type _Tlen struct{}

var _getLen = _Tlen{}

func jGet(v interface{}, keys ...interface{}) interface{} {
	for _, key := range keys {
		switch k := key.(type) {
		case string:
			obj, ok := v.(map[string]interface{})
			if !ok {
				return nil
			}
			v = obj[k]
		case int:
			obj, ok := v.([]interface{})
			if !ok {
				return nil
			}
			if k < 0 {
				return nil
			}
			if k >= len(obj) {
				return nil
			}
			v = obj[k]
		case _Tlen:
			obj, ok := v.([]interface{})
			if !ok {
				return nil
			}
			return len(obj)
		default:
			return nil
		}
	}
	return v
}

func jObj(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	obj, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return obj
}

func jStr(v interface{}) string {
	if v == nil {
		return ""
	}
	str, ok := v.(string)
	if !ok {
		return ""
	}
	return str
}

var colorMap = map[string]string{
	"topBlack":     "#24292e",
	"pushBlue":     "#052049",
	"approveGreen": "#2cbe4e",
	"mergePurple":  "#6f42c1",
	"rejectRed":    "#cb2431",
}

func (mod *GithookModule) RenderPush(payload interface{}) slack.OutgoingSlackMessage {
	/*
		[riking/marvin] *riking* pushed 2 new commits to master:
	*/
	var msg slack.OutgoingSlackMessage
	var atch slack.Attachment
	var buf bytes.Buffer
	msg.UnfurlLinks = util.TriNo
	msg.LinkNames = util.TriNo
	msg.Parse = slack.ParseStyleNone
	atch.AuthorIcon = "https://assets-cdn.github.com/favicon.ico"
	atch.AuthorName = fmt.Sprintf("[%s]", jGet(payload, "repository", "full_name"))
	atch.AuthorLink = jStr(jGet(payload, "repository", "html_url"))
	atch.TS = time.Now().Unix()
	atch.Color = colorMap["pushBlue"]
	tsFloat, ok := jGet(payload, "repository", "pushed_at").(float64)
	if ok {
		atch.TS = int64(tsFloat)
	}

	compare := jStr(jGet(payload, "compare"))
	ref := jStr(jGet(payload, "ref"))
	ref = strings.TrimLeft(ref, "refs/heads/")
	commits, _ := jGet(payload, "commits").([]interface{})

	fmt.Fprintf(&buf, "*<%s|%s>* pushed <%s|%v new commits> to *%s*\n",
		jStr(jGet(payload, "sender", "html_url")), atcommand.SanitizeAt(jStr(jGet(payload, "sender", "login"))),
		compare, len(commits), atcommand.SanitizeAt(ref))

	apiURL := strings.Replace(compare, "github.com/", "api.github.com/repos/", 1)
	fmt.Fprintf(&buf, "%s\n", getDiffStat(apiURL))

	for idx, commit := range commits {
		if idx == 3 {
			fmt.Fprintf(&buf, "… %d more commits", len(commits)-3)
			break
		}
		objid := jStr(jGet(commit, "id"))
		timestamp := jStr(jGet(commit, "timestamp"))
		ts, err := time.Parse("2006-01-02T15:04:05Z07:00", timestamp)
		if err != nil {
			util.LogError(err)
		}
		commitMsg := jStr(jGet(commit, "message"))
		idx := strings.Index(commitMsg, "\n")
		if idx != -1 {
			commitMsg = commitMsg[:idx]
		}
		fmt.Fprintf(&buf, "<%s|%s> by *%s* [<!date^%d^{time}|%s>] %s\n",
			jStr(jGet(commit, "url")), objid[:8], atcommand.SanitizeAt(jStr(jGet(commit, "author", "name"))),
			ts.Unix(), ts.Format(time.Kitchen),
			atcommand.SanitizeAt(commitMsg))
	}

	atch.Text = buf.String()
	msg.Attachments = []slack.Attachment{atch}
	return msg
}

var verbMap = map[string]string{
	"review_requested":       "requested review on",
	"review_request_removed": "removed review request on",
	"synchronize":            "updated",
	"edited":                 "edited description of",
}

var prColors = map[string]string{
	"":         "topBlack",
	"opened":   "approveGreen",
	"closed":   "rejectRed",
	"reopened": "topBlack",
	"updated":  "pushBlue",
}

func (mod *GithookModule) RenderPR(payload interface{}) slack.OutgoingSlackMessage {
	var msg slack.OutgoingSlackMessage
	var atch slack.Attachment
	var buf bytes.Buffer
	msg.UnfurlLinks = util.TriNo
	msg.LinkNames = util.TriNo
	msg.Parse = slack.ParseStyleNone
	atch.AuthorIcon = "https://assets-cdn.github.com/favicon.ico"
	atch.AuthorName = fmt.Sprintf("[%s]", jGet(payload, "repository", "full_name"))
	atch.AuthorLink = jStr(jGet(payload, "pull_request", "html_url"))
	atch.TS = time.Now().Unix()
	atch.Color = colorMap["topBlack"]

	author := jStr(jGet(payload, "sender", "login"))
	verb := jStr(jGet(payload, "action"))
	if verbMap[verb] != "" {
		verb = verbMap[verb]
	}
	if prColors[verb] != "" {
		atch.Color = colorMap[prColors[verb]]
	}
	if verb == "closed" {
		merged, _ := jGet(payload, "pull_request", "merged").(bool)
		if merged {
			atch.Color = colorMap["mergePurple"]
		}
	}

	fmt.Fprintf(&buf, "*<%s|%s>* %s <%s|PR #%v> (%s...%s): %s",
		jGet(payload, "sender", "html_url"), atcommand.SanitizeAt(author),
		verb,
		jGet(payload, "pull_request", "html_url"), jGet(payload, "number"),
		jGet(payload, "pull_request", "base", "ref"), jGet(payload, "pull_request", "head", "ref"),
		jGet(payload, "pull_request", "title"),
	)
	atch.Fallback = fmt.Sprintf("[%s] %s %s PR #%v",
		jGet(payload, "repository", "full_name"),
		jGet(payload, "sender", "login"), verb,
		jGet(payload, "number"), jGet(payload, "pull_request", "title"))
	if verb == "opened" || verb == "edited description of" {
		prBody := jStr(jGet(payload, "pull_request", "body"))
		if prBody != "" {
			fmt.Fprintf(&buf, "\n%s", prBody)
		}
	}
	atch.Text = buf.String()
	msg.Attachments = []slack.Attachment{atch}
	return msg
}

func (mod *GithookModule) RenderComment(payload interface{}) slack.OutgoingSlackMessage {
	var msg slack.OutgoingSlackMessage
	var atch slack.Attachment
	var buf bytes.Buffer
	msg.UnfurlLinks = util.TriNo
	msg.LinkNames = util.TriNo
	msg.Parse = slack.ParseStyleNone
	atch.AuthorIcon = "https://assets-cdn.github.com/favicon.ico"
	atch.AuthorName = fmt.Sprintf("[%s]", jGet(payload, "repository", "full_name"))
	atch.AuthorLink = jStr(jGet(payload, "comment", "html_url"))
	atch.TS = time.Now().Unix()
	atch.Color = colorMap["topBlack"]

	author := jStr(jGet(payload, "sender", "login"))
	verb := jStr(jGet(payload, "action"))
	if jStr(jGet(payload, "issue", "state")) == "closed" {
		atch.Color = colorMap["rejectRed"]
	}
	fmt.Fprintf(&buf, "*<%s|%s>* %s comment on <%s|#%d>: *%s*",
		jGet(payload, "sender", "html_url"), atcommand.SanitizeAt(author),
		verb,
		jGet(payload, "comment", "html_url"), jGet(payload, "issue", "number"),
		jGet(payload, "issue", "title"),
	)
	if verb == "created" {
		fmt.Fprintf(&buf, "\n%s", jGet(payload, "comment", "body"))
	}
	atch.Fallback = fmt.Sprintf("%s %s comment on #%v: %s",
		author, verb, jGet(payload, "number"), jGet(payload, "issue", "title"))
	atch.Text = buf.String()
	msg.Attachments = []slack.Attachment{atch}
	return msg
}

func (mod *GithookModule) RenderPRReview(payload interface{}) slack.OutgoingSlackMessage {
	var msg slack.OutgoingSlackMessage
	var atch slack.Attachment
	var buf bytes.Buffer
	msg.UnfurlLinks = util.TriNo
	msg.LinkNames = util.TriNo
	msg.Parse = slack.ParseStyleNone
	atch.AuthorIcon = "https://assets-cdn.github.com/favicon.ico"
	atch.AuthorName = fmt.Sprintf("[%s]", jGet(payload, "repository", "full_name"))
	atch.AuthorLink = jStr(jGet(payload, "review", "html_url"))
	atch.TS = time.Now().Unix()
	atch.Color = colorMap["topBlack"]

	author := jStr(jGet(payload, "sender", "login"))
	verb := jStr(jGet(payload, "action"))

	if jGet(payload, "submitted_at") == nil {
		return slack.OutgoingSlackMessage{}
	}
	fmt.Fprintf(&buf, "*<%s|%s>* %s review on <%s|#%v> (%s...%s): *%s*\n<%s>",
		jGet(payload, "sender", "html_url"), author,
		verb,
		jGet(payload, "review", "html_url"), jGet(payload, "number"),
		jGet(payload, "pull_request", "base", "ref"), jGet(payload, "pull_request", "head", "ref"),
		jGet(payload, "pull_request", "title"),
		jGet(payload, "review", "html_url"),
	)
	if verb == "submitted" && jGet(payload, "review", "body") != nil {
		fmt.Fprintf(&buf, "\n%s", jGet(payload, "review", "body"))
	}
	if verb == "submitted" {
		switch jStr(jGet("review", "state")) {
		case "approved", "APPROVED":
			atch.Color = colorMap["approveGreen"]
		case "comment", "COMMENT":
			atch.Color = colorMap["pushBlue"]
		case "request_changes", "REQUEST_CHANGES":
			atch.Color = colorMap["rejectRed"]
		}
	}
	atch.Fallback = fmt.Sprintf("%s %s review on #%v: %s",
		author, verb, jGet(payload, "number"), jGet(payload, "pull_request", "title"))
	atch.Text = buf.String()
	msg.Attachments = []slack.Attachment{atch}
	return msg
}

func getDiffStat(apiURL string) string {
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Sprintf("(Error getting diffstat: %s)", err)
	}
	defer resp.Body.Close()
	var response struct {
		Files []struct {
			Additions int
			Deletions int
			Status    string
		}
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return fmt.Sprintf("(Error getting diffstat: %s)", err)
	}
	var totalAdd = 0
	var totalDel = 0
	var fileAdd = 0
	var fileDel = 0
	var fileMod = 0
	for _, v := range response.Files {
		totalAdd += v.Additions
		totalDel += v.Deletions
		switch v.Status {
		case "modified":
			fileMod++
		case "added":
			fileAdd++
		case "removed":
			fileDel++
		}
	}
	if fileAdd > 0 || fileDel > 0 {
		return fmt.Sprintf("[++%d --%d] %d files changed, %d added, %d removed", totalAdd, totalDel, fileMod, fileAdd, fileDel)
	}
	return fmt.Sprintf("[++%d --%d] %d files changed", totalAdd, totalDel, fileMod)
}
