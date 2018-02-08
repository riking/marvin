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

func (mod *GithookModule) RenderPush(payload interface{}) slack.OutgoingSlackMessage {
	/*
		[riking/marvin] *riking* pushed 2 new commits to master:
	*/
	var msg slack.OutgoingSlackMessage
	msg.UnfurlLinks = util.TriNo
	msg.LinkNames = util.TriNo
	msg.Parse = slack.ParseStyleNone

	var atch slack.Attachment
	atch.AuthorIcon = "https://assets-cdn.github.com/favicon.ico"
	atch.AuthorName = fmt.Sprintf("[%s] push", jGet(payload, "repository", "full_name"))
	atch.Color = "#24292e"
	tsFloat, ok := jGet(payload, "repository", "pushed_at").(float64)
	if ok {
		atch.TS = int64(tsFloat)
	}
	var buf bytes.Buffer

	compare := jStr(jGet(payload, "compare"))
	if compare == "" {
		return slack.OutgoingSlackMessage{}
	}
	atch.AuthorLink = compare

	ref := jStr(jGet(payload, "ref"))
	ref = strings.TrimLeft(ref, "refs/heads/")
	commits, _ := jGet(payload, "commits").([]interface{})

	fmt.Fprintf(&buf, "*<%s|%s>* pushed <%s|%d new commits> to *%s*\n",
		jStr(jGet(payload, "sender", "url")), atcommand.SanitizeAt(jStr(jGet(payload, "sender", "login"))),
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
		fmt.Fprintf(&buf, "<%s|%s> by *%s* [<!date^%d^{time}|%s>] %s\n",
			jStr(jGet(commit, "url")), objid[:8], atcommand.SanitizeAt(jStr(jGet(commit, "author", "name"))),
			ts.Unix(), ts.Format(time.Kitchen),
			atcommand.SanitizeAt(jStr(jGet(commit, "message"))))
	}

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
		return fmt.Sprintf("[+%d -%d] %d files changed, %d added, %d removed", totalAdd, totalDel, fileMod, fileAdd, fileDel)
	}
	return fmt.Sprintf("[+%d -%d] %d files changed", totalAdd, totalDel, fileMod)
}
