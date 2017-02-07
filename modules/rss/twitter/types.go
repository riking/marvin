package twitter

type Tweet struct {
	Coordinates *struct {
		Coordinates [2]float64 `json:"coordinates"`
		Type        string     `json:"type"`
	} `json:"coordinates"`
	CreatedAt string `json:"created_at"`
	// CurrentUserRetweet   *TweetIdentifier       `json:"current_user_retweet"`
	Entities struct {
		Hashtags     []HashtagEntity `json:"hashtags"`
		Media        []MediaEntity   `json:"media"`
		Urls         []URLEntity     `json:"urls"`
		UserMentions []MentionEntity `json:"user_mentions"`
	} `json:"entities"`
	FavoriteCount        int                    `json:"favorite_count"`
	Favorited            bool                   `json:"favorited"`
	FilterLevel          string                 `json:"filter_level"`
	ID                   int64                  `json:"id"`
	IDStr                string                 `json:"id_str"`
	InReplyToScreenName  string                 `json:"in_reply_to_screen_name"`
	InReplyToStatusID    int64                  `json:"in_reply_to_status_id"`
	InReplyToStatusIDStr string                 `json:"in_reply_to_status_id_str"`
	InReplyToUserID      int64                  `json:"in_reply_to_user_id"`
	InReplyToUserIDStr   string                 `json:"in_reply_to_user_id_str"`
	Lang                 string                 `json:"lang"`
	PossiblySensitive    bool                   `json:"possibly_sensitive"`
	RetweetCount         int                    `json:"retweet_count"`
	Retweeted            bool                   `json:"retweeted"`
	RetweetedStatus      *Tweet                 `json:"retweeted_status"`
	Source               string                 `json:"source"`
	Scopes               map[string]interface{} `json:"scopes"`
	Text                 string                 `json:"text"`
	Place                *Place                 `json:"place"`
	Truncated            bool                   `json:"truncated"`
	User                 *User                  `json:"user"`
	WithheldCopyright    bool                   `json:"withheld_copyright"`
	WithheldInCountries  []string               `json:"withheld_in_countries"`
	WithheldScope        string                 `json:"withheld_scope"`
	ExtendedEntities     *ExtendedEntity        `json:"extended_entities"`
	QuotedStatusID       int64                  `json:"quoted_status_id"`
	QuotedStatusIDStr    string                 `json:"quoted_status_id_str"`
	QuotedStatus         *Tweet                 `json:"quoted_status"`
}

type Entities struct {
	Hashtags     []HashtagEntity `json:"hashtags"`
	Media        []MediaEntity   `json:"media"`
	Urls         []URLEntity     `json:"urls"`
	UserMentions []MentionEntity `json:"user_mentions"`
}

type ExtendedEntity struct {
	Media []MediaEntity `json:"media"`
}

type TweetIdentifier struct {
	ID    int64  `json:"id"`
	IDStr string `json:"id_str"`
}

type Indices [2]int

type URLEntity struct {
	Indices     Indices `json:"indices"`
	DisplayURL  string  `json:"display_url"`
	ExpandedURL string  `json:"expanded_url"`
	URL         string  `json:"url"`
}

type MentionEntity struct {
	Indices    Indices `json:"indices"`
	ID         int64   `json:"id"`
	IDStr      string  `json:"id_str"`
	Name       string  `json:"name"`
	ScreenName string  `json:"screen_name"`
}

type HashtagEntity struct {
	Indices Indices `json:"indices"`
	Text    string  `json:"text"`
}

type MediaEntity struct {
	URLEntity
	ID                int64  `json:"id"`
	IDStr             string `json:"id_str"`
	MediaURL          string `json:"media_url"`
	MediaURLHttps     string `json:"media_url_https"`
	SourceStatusID    int64  `json:"source_status_id"`
	SourceStatusIDStr string `json:"source_status_id_str"`
	Type              string `json:"type"`
	Sizes             struct {
		Thumb  MediaSize `json:"thumb"`
		Large  MediaSize `json:"large"`
		Medium MediaSize `json:"medium"`
		Small  MediaSize `json:"small"`
	} `json:"sizes"`
	VideoInfo struct {
		AspectRatio    [2]int `json:"aspect_ratio"`
		DurationMillis int    `json:"duration_millis"`
		Variants       []struct {
			ContentType string `json:"content_type"`
			Bitrate     int    `json:"bitrate"`
			URL         string `json:"url"`
		} `json:"variants"`
	} `json:"video_info"`
}

type MediaSize struct {
	Width  int    `json:"w"`
	Height int    `json:"h"`
	Resize string `json:"resize"`
}

type BoundingBox struct {
	Coordinates [][][2]float64 `json:"coordinates"`
	Type        string         `json:"type"`
}

type Place struct {
	Attributes  map[string]string `json:"attributes"`
	BoundingBox *BoundingBox      `json:"bounding_box"`
	Country     string            `json:"country"`
	CountryCode string            `json:"country_code"`
	FullName    string            `json:"full_name"`
	Geometry    *BoundingBox      `json:"geometry"`
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	PlaceType   string            `json:"place_type"`
	Polylines   []string          `json:"polylines"`
	URL         string            `json:"url"`
}

type User struct {
	ContributorsEnabled bool   `json:"contributors_enabled"`
	CreatedAt           string `json:"created_at"`
	DefaultProfile      bool   `json:"default_profile"`
	DefaultProfileImage bool   `json:"default_profile_image"`
	Description         string `json:"description"`
	Email               string `json:"email"`
	Entities            struct {
		URL         Entities `json:"url"`
		Description Entities `json:"description"`
	} `json:"entities"`
	FavouritesCount                int      `json:"favourites_count"`
	FollowRequestSent              bool     `json:"follow_request_sent"`
	Following                      bool     `json:"following"`
	FollowersCount                 int      `json:"followers_count"`
	FriendsCount                   int      `json:"friends_count"`
	GeoEnabled                     bool     `json:"geo_enabled"`
	ID                             int64    `json:"id"`
	IDStr                          string   `json:"id_str"`
	IsTranslator                   bool     `json:"is_translator"`
	Lang                           string   `json:"lang"`
	ListedCount                    int      `json:"listed_count"`
	Location                       string   `json:"location"`
	Name                           string   `json:"name"`
	Notifications                  bool     `json:"notifications"`
	ProfileBackgroundColor         string   `json:"profile_background_color"`
	ProfileBackgroundImageURL      string   `json:"profile_background_image_url"`
	ProfileBackgroundImageURLHttps string   `json:"profile_background_image_url_https"`
	ProfileBackgroundTile          bool     `json:"profile_background_tile"`
	ProfileBannerURL               string   `json:"profile_banner_url"`
	ProfileImageURL                string   `json:"profile_image_url"`
	ProfileImageURLHttps           string   `json:"profile_image_url_https"`
	ProfileLinkColor               string   `json:"profile_link_color"`
	ProfileSidebarBorderColor      string   `json:"profile_sidebar_border_color"`
	ProfileSidebarFillColor        string   `json:"profile_sidebar_fill_color"`
	ProfileTextColor               string   `json:"profile_text_color"`
	ProfileUseBackgroundImage      bool     `json:"profile_use_background_image"`
	Protected                      bool     `json:"protected"`
	ScreenName                     string   `json:"screen_name"`
	ShowAllInlineMedia             bool     `json:"show_all_inline_media"`
	Status                         *Tweet   `json:"status"`
	StatusesCount                  int      `json:"statuses_count"`
	Timezone                       string   `json:"time_zone"`
	URL                            string   `json:"url"`
	UtcOffset                      int      `json:"utc_offset"`
	Verified                       bool     `json:"verified"`
	WithheldInCountries            []string `json:"withheld_in_countries"`
	WithholdScope                  string   `json:"withheld_scope"`
}
