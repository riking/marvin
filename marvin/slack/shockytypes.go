package slack

import "encoding/json"

type RTMRawMessage map[string]interface{}

const MsgFieldRawBytes = "_rawBytes"

func (m RTMRawMessage) Type() string         { return m["type"].(string) }
func (m RTMRawMessage) Okay() bool           { q, _ := m["ok"].(bool); return q }
func (m RTMRawMessage) Original() []byte     { q, _ := m[MsgFieldRawBytes].([]byte); return q }
func (m RTMRawMessage) Subtype() string      { q, _ := m["subtype"].(string); return q }
func (m RTMRawMessage) ChannelID() ChannelID { q, _ := m["channel"].(string); return ChannelID(q) }
func (m RTMRawMessage) UserID() UserID       { q, _ := m["user"].(string); return UserID(q) }
func (m RTMRawMessage) Text() string         { q, _ := m["text"].(string); return q }
func (m RTMRawMessage) Timestamp() MessageTS { q, _ := m["ts"].(string); return MessageTS(q) }
func (m RTMRawMessage) IsHidden() bool       { q, _ := m["hidden"].(bool); return q }

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
	Domain      string       `json:"domain,omitempty"`
	ChannelID   ChannelID    `json:"channel"`
	Username    UserID       `json:"username"`
	Text        string       `json:"text"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	IconURL     string       `json:"icon_url,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	UnfurlLinks bool         `json:"unfurl_links,omitempty"`
	Parse       ParseStyle   `json:"parse,omitempty"`
	LinkNames   bool         `json:"link_names,omitempty"`
	Markdown    bool         `json:"mrkdwn,omitempty"`
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
	MarkdownIn []MarkdownField   `json:"mrkdown_in,omitempty"`
}

type MarkdownField string

var (
	MarkdownFieldPretext  = MarkdownField("pretext")
	MarkdownFieldText     = MarkdownField("text")
	MarkdownFieldTitle    = MarkdownField("title")
	MarkdownFieldFields   = MarkdownField("fields")
	MarkdownFieldFallback = MarkdownField("fallback")
)

type AttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}
