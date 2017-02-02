package slack

import (
	"encoding/json"

	"github.com/riking/marvin/util"
)

type RTMRawMessage map[string]interface{}

const MsgFieldRawBytes = "_rawBytes"

func (m RTMRawMessage) Type() string         { return m["type"].(string) }
func (m RTMRawMessage) Okay() bool           { q, _ := m["ok"].(bool); return q }
func (m RTMRawMessage) Original() []byte     { q, _ := m[MsgFieldRawBytes].([]byte); return q }
func (m RTMRawMessage) Subtype() string      { q, _ := m["subtype"].(string); return q }
func (m RTMRawMessage) ChannelID() ChannelID { q, _ := m["channel"].(string); return ChannelID(q) }
func (m RTMRawMessage) UserID() UserID       { q, _ := m["user"].(string); return UserID(q) }
func (m RTMRawMessage) Text() string         { q, _ := m["text"].(string); return q }
func (m RTMRawMessage) MessageTS() MessageTS { q, _ := m["ts"].(string); return MessageTS(q) }
func (m RTMRawMessage) EventTS() MessageTS   { q, _ := m["ts"].(string); return MessageTS(q) }
func (m RTMRawMessage) IsHidden() bool       { q, _ := m["hidden"].(bool); return q }
func (m RTMRawMessage) MessageID() MessageID {
	return MessageID{ChannelID: m.ChannelID(), MessageTS: m.MessageTS()}
}

func (m RTMRawMessage) StringField(field string) string {
	q, _ := m[field].(string)
	return q
}

func (m RTMRawMessage) ReplyTo() int {
	q, _ := m["reply_to"].(float64)
	return int(q)
}

func (m RTMRawMessage) Attachments() []Attachment {
	panic("NotImplemented")
}

func (m RTMRawMessage) ReMarshal(v interface{}) error {
	return json.Unmarshal(m.Original(), v)
}

func (m RTMRawMessage) String() string {
	bytes, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func (m RTMRawMessage) AssertText() bool {
	return m.Type() == "message" && m.Subtype() != "message_changed" && m.Subtype() != "message_deleted"
}

type EditMessage struct {
	RTMRawMessage
}

func (m EditMessage) EditHash() map[string]interface{} {
	q, _ := m.RTMRawMessage["message"].(map[string]interface{})
	return q
}
func (m EditMessage) EditingUserID() UserID {
	q1, _ := m.EditHash()["edited"].(map[string]interface{})
	if q1 == nil {
		return ""
	}
	q2, _ := q1["user"].(string)
	return UserID(q2)
}
func (m EditMessage) UserID() UserID        { q, _ := m.EditHash()["user"].(string); return UserID(q) }
func (m EditMessage) MessageUserID() UserID { q, _ := m.EditHash()["user"].(string); return UserID(q) }
func (m EditMessage) Text() string          { q, _ := m.EditHash()["text"].(string); return q }
func (m EditMessage) MessageTS() MessageTS  { q, _ := m.EditHash()["ts"].(string); return MessageTS(q) }
func (m EditMessage) EventTS() MessageTS    { q, _ := m.RTMRawMessage["ts"].(string); return MessageTS(q) }
func (m EditMessage) Subtype() string       { return "" }
func (m EditMessage) MessageID() MessageID {
	return MessageID{ChannelID: m.ChannelID(), MessageTS: m.MessageTS()}
}
func (m EditMessage) AssertText() bool {
	return m.RTMRawMessage.Type() == "message" && m.RTMRawMessage.Subtype() == "message_changed"
}

type SlackTextMessage interface {
	UserID() UserID
	ChannelID() ChannelID
	MessageID() MessageID
	MessageTS() MessageTS
	EventTS() MessageTS
	Subtype() string
	Text() string
	AssertText() bool
}

type IncomingReaction struct {
	Type       string
	IsRemoval  bool
	UserID     UserID
	Reaction   string
	ItemUserID UserID
	Item       struct {
		Type string
		// type = message
		MsgChannelID ChannelID
		MsgTS        MessageTS
		// type = file
		FileID FileID
		// type = file_comment
		FileCommentID FileCommentID
	}
}

type ParseStyle string

const (
	ParseStyleFull = ParseStyle("full")
	ParseStyleNone = ParseStyle("none")
)

type OutgoingSlackMessage struct {
	Text        string        `json:"text,omitempty"`
	Attachments []Attachment  `json:"attachments,omitempty"`
	UnfurlLinks util.TriValue `json:"unfurl_links,omitempty"`
	Parse       ParseStyle    `json:"parse,omitempty"`
	LinkNames   util.TriValue `json:"link_names,omitempty"`
	Markdown    util.TriValue `json:"mrkdwn,omitempty"`
}

type SlashCommandRequest struct {
	Token       string
	TeamId      TeamID    `schema:"team_id"`
	TeamDomain  string    `schema:"team_domain"` // no .slack.com
	ChannelId   ChannelID `schema:"channel_id"`
	ChannelName string    `schema:"channel_name"`
	UserId      UserID    `schema:"user_id"`
	UserName    string    `schema:"user_name"`
	Command     string
	Text        string
	Msg         RTMRawMessage `schema:"-"`
	ResponseURL string        `schema:"response_url"`
}

type ResponseType string

const (
	ResponseTypeEphermal  ResponseType = "ephemeral"
	ResponseTypeInChannel              = "in_channel"
	ResponseTypeDelayed                = "__delayed"
)

type SlashCommandResponse struct {
	OutgoingSlackMessage
	ResponseType ResponseType `json:"response_type,omitempty"`
}

type Attachment struct {
	Fallback   string            `json:"fallback"`
	Pretext    string            `json:"pretext,omitempty"`
	Text       string            `json:"text,omitempty"`
	Color      string            `json:"color,omitempty"`
	Fields     []AttachmentField `json:"fields,omitempty"`
	AuthorName string            `json:"author_name,omitempty"`
	AuthorLink string            `json:"author_link,omitempty"`
	AuthorIcon string            `json:"author_icon,omitempty"`
	Title      string            `json:"title,omitempty"`
	TitleLink  string            `json:"title_link,omitempty"`
	TS         int64             `json:"ts,omitempty"`
	ImageURL   string            `json:"image_url,omitempty"`
	Footer     string            `json:"footer,omitempty"`
}

type AttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}
