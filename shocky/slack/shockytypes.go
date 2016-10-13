package slack

type MessageSubtype string

const (
	MessageSubtypeBotMessage     MessageSubtype = "bot_message"
	MessageSubtypeChannelJoin                   = "channel_join"
	MessageSubtypeChannelLeave                  = "channel_leave"
	MessageSubtypeMeMessage                     = "me_message"
	MessageSubtypeMessageChanged                = "message_changed"
	MessageSubtypeMessageDeleted                = "message_deleted"
)

type RTMRawMessage map[string]interface{}

func (m RTMRawMessage) Type() string         { return m["type"].(string) }
func (m RTMRawMessage) Subtype() string      { q, _ := m["subtype"].(string); return q }
func (m RTMRawMessage) ChannelID() ChannelID { q, _ := m["channel"].(string); return ChannelID(q) }
func (m RTMRawMessage) UserID() UserID       { q, _ := m["user"].(string); return UserID(q) }
func (m RTMRawMessage) Text() string         { q, _ := m["text"].(string); return q }
func (m RTMRawMessage) Timestamp() string    { q, _ := m["ts"].(string); return q }
func (m RTMRawMessage) IsHidden() bool       { q, _ := m["hidden"].(bool); return q }
func (m RTMRawMessage) Attachments() []Attachment {
	panic("NotImplemented")
}

type IncomingMessage struct {
	ChannelID  ChannelID
	UserID     UserID
	Text       string
	Subtype    string
	RawMessage RTMRawMessage
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
		MsgTS        string
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

type Message struct {
	Domain      string       `json:"domain"`
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
	Msg         IncomingMessage `schema:"-"`
	ResponseURL string          `schema:"response_url"`
}

type ResponseType string

const (
	ResponseTypeEphermal  ResponseType = "ephemeral"
	ResponseTypeInChannel              = "in_channel"
	ResponseTypeDelayed                = "__delayed"
)

type SlashCommandResponse struct {
	Message
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
