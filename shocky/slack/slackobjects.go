package slack

import "fmt"

type TeamID string
type UserID string
type ChannelID string
type FileID string
type FileCommentID string

type APIResponse struct {
	OK bool `json:"ok"`
	SlackError string `json:"error"`
	Warning string `json:"warning"`
}

func (r *APIResponse) Error() string {
	var w string = ""
	if r.Warning != "" {
		w = fmt.Sprintf(" (warning: %s)", r.Warning)
	}
	return fmt.Sprintf("slack API error: %s%s", r.SlackError, w)
}

type TeamInfo struct {
	ID                TeamID
	Name              string
	Domain            string
	EmailDomain       string `json:"email_domain"`
	Icon              interface{}
	MsgEditWindowMins float64 `json:"msg_edit_window_mins"`
	OverStorageLimit  bool    `json:"over_storage_limit"`
	Prefs             interface{}
	Plan              string
}

type User struct {
	ID                UserID
	Name              string
	Deleted           bool
	Color             string
	IsAdmin           bool   `json:"is_admin"`
	IsOwner           bool   `json:"is_owner"`
	IsPrimaryOwner    bool   `json:"is_primary_owner"`
	IsRestricted      bool   `json:"is_restricted"`
	IsUltraRestricted bool   `json:"is_ultra_restricted"`
	Has2FA            bool   `json:"has_2fa"`
	TwoFactorType     string `json:"two_factor_type"`
	HasFiles          bool   `json:"has_files"`
	Profile           struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		RealName  string `json:"real_name"`
		Email     string
		Skype     string
		Phone     string
		Image72   string
	}
}

type Channel struct {
	ID         ChannelID
	Name       string
	IsChannel  interface{} `json:"is_channel"`
	IsGroup    interface{} `json:"is_group"`
	IsMPIM     interface{} `json:"is_mpim"`
	IsIM       bool        `json:"is_im"`
	Created    float64     // unix millis
	Creator    UserID
	IsArchived bool `json:"is_archived"`
	IsGeneral  bool `json:"is_general"`
	// IM only
	IsUserDeleted bool `json:"is_user_deleted"`
	Members       []UserID
	IsMember      bool `json:"is_member"`
	// IM only
	IsOpen   bool   `json:"is_open"`
	LastRead string `json:"last_read"`
	Latest   []interface{}
	Topic    struct {
		Value   string
		Creator UserID
		LastSet float64 `json:"last_set"`
	}
	Purpose struct {
		Value   string
		Creator UserID
		LastSet float64 `json:"last_set"`
	}
}

func (c *Channel) IsPublicChannel() bool {
	str, ok := c.IsChannel.(string)
	if ok {
		return str == "true"
	}
	b, ok := c.IsChannel.(bool)
	if ok {
		return b
	}
	return false
}

func (c *Channel) IsPrivateChannel() bool {
	str, ok := c.IsGroup.(string)
	if ok {
		return str == "true"
	}
	b, ok := c.IsGroup.(bool)
	if ok {
		return b
	}
	return false
}

func (c *Channel) IsMultiIM() bool {
	str, ok := c.IsMPIM.(string)
	if ok {
		return str == "true"
	}
	b, ok := c.IsMPIM.(bool)
	if ok {
		return b
	}
	return false
}

func (c *Channel) IsPrivateMessage() bool {
	return c.IsIM
}
