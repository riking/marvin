package slack

import (
	"fmt"
	"regexp"
	"strings"
)

// https://github.com/slack-ruby/slack-ruby-client/blob/master/lib/slack/messages/formatting.rb

var entityRgx = regexp.MustCompile(`<([?@#!]?)(.*?)>`)

// UnescapeText unwraps URLs in a Slack message and otherwise canonicalizes
// certain entities for use by Marvin.
//
// Notably, &lt; and &gt; are left alone to prevent someone from typing <!everyone> and having Marvin repeat it as a @everyone.
func UnescapeText(msg string) string {
	msg = strings.Replace(msg, "“", "\"", -1)
	msg = strings.Replace(msg, "”", "\"", -1)
	msg = strings.Replace(msg, "‘", "'", -1)
	msg = strings.Replace(msg, "’", "'", -1)

	msg = entityRgx.ReplaceAllStringFunc(msg, func(entity string) string {
		match := entityRgx.FindStringSubmatch(entity)
		rhsSplit := strings.SplitN(match[2], "|", 2)
		mid := ""
		rhs := rhsSplit[0]
		if len(rhsSplit) == 2 {
			mid = rhsSplit[0]
			rhs = rhsSplit[1]
		}
		switch match[1] {
		case "@":
			return fmt.Sprintf("<@%s>", rhsSplit[0])
		case "!":
			if strings.HasPrefix(mid, "date") {
				return rhs
			}
			if strings.HasPrefix(mid, "subteam") {
				return entity
			}
			return fmt.Sprintf("@/%s", rhs)
		case "#":
			return entity
		default:
			if strings.HasPrefix(mid, "mailto") {
				return rhs
			}
			if strings.HasPrefix(mid, "http") {
				return rhs
			}
			return rhs
		}
	})
	// < and > are left alone
	msg = strings.Replace(msg, "&amp;", "&", -1)
	return msg
}
