package slack

import (
	"fmt"
	"time"
)

type TeamID string
type EnterpriseID string
type UserID string
type ChannelID string
type FileID string
type FileCommentID string
type MessageTS string

const MessageTSCharsAfterDot = 6

func (u UserID) Raw() string      { return string(u) }
func (u UserID) ToAtForm() string { return fmt.Sprintf("<@%s>", string(u)) }

func (u UserID) Format(f fmt.State, c rune) {
	if c == 'v' {
		fmt.Fprintf(f, "<@%s>", string(u))
	} else {
		fmt.Fprint(f, string(u))
	}
}

type MessageID struct {
	ChannelID
	MessageTS
}

func MsgID(ch ChannelID, ts MessageTS) MessageID { return MessageID{ch, ts} }

type APIResponse struct {
	OK         bool   `json:"ok"`
	SlackError string `json:"error"`
	Warning    string `json:"warning"`
}

func (r APIResponse) Error() string {
	var w string = ""
	if r.Warning != "" {
		w = fmt.Sprintf(" (warning: %s)", r.Warning)
	}
	if r.OK {
		return "OK" + w
	}
	return fmt.Sprintf("slack API error: %s%s", r.SlackError, w)
}

type CodedError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (ce CodedError) Error() string {
	return ce.Msg
}

type LatestMsg struct {
	User    string    `json:"user"`
	Text    string    `json:"text"`
	Type    string    `json:"type"`
	Subtype string    `json:"subtype"`
	Ts      MessageTS `json:"ts"`
}

type TeamInfo struct {
	ID                    TeamID `json:"id"`
	Name                  string
	Domain                string
	EmailDomain           string `json:"email_domain"`
	Icon                  interface{}
	MsgEditWindowMins     float64 `json:"msg_edit_window_mins"`
	OverStorageLimit      bool    `json:"over_storage_limit"`
	Prefs                 interface{}
	Plan                  string
	AvatarBaseURL         string `json:"avatar_base_url"`
	OverIntegrationsLimit bool   `json:"over_integrations_limit"`
}

type User struct {
	NotExist bool      `json:"-"`
	CacheTS  time.Time `json:"-"`

	ID                UserID      `json:"id"`
	TeamID            TeamID      `json:"team_id"`
	Name              string      `json:"-"`
	Deleted           bool        `json:"deleted"`
	Status            interface{} `json:"status"`
	Color             string      `json:"color"`
	RealName          string      `json:"real_name"`
	Tz                string      `json:"tz"`
	TzLabel           string      `json:"tz_label"`
	TzOffset          int         `json:"tz_offset"`
	Updated           int64       `json:"updated"`
	Profile           Profile     `json:"profile"`
	IsAdmin           bool        `json:"is_admin"`
	IsOwner           bool        `json:"is_owner"`
	IsPrimaryOwner    bool        `json:"is_primary_owner"`
	IsRestricted      bool        `json:"is_restricted"`
	IsUltraRestricted bool        `json:"is_ultra_restricted"`
	IsBot             bool        `json:"is_bot"`
	Presence          string      `json:"presence"`
	Has2Fa            bool        `json:"has_2fa,omitempty"`
}

type Profile struct {
	DisplayName        string `json:"display_name_normalized"`
	RealName           string `json:"real_name"`
	RealNameNormalized string `json:"real_name_normalized"`
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	Email              string `json:"email"`
	Phone              string `json:"phone"`
	Title              string `json:"title"`
	Skype              string `json:"skype"`
	Image24            string `json:"image_24"`
	Image32            string `json:"image_32"`
	Image48            string `json:"image_48"`
	Image72            string `json:"image_72"`
	Image128           string `json:"image_128"`
	Image192           string `json:"image_192"`
	Image512           string `json:"image_512"`
	Image1024          string `json:"image_1024"`
	ImageOriginal      string `json:"image_original"`
}

func (u *User) Avatar(size int) string {
	fields := []struct {
		Size  int
		Value string
	}{
		{24, u.Profile.Image24},
		{32, u.Profile.Image32},
		{48, u.Profile.Image48},
		{72, u.Profile.Image72},
		{192, u.Profile.Image192},
		{512, u.Profile.Image512},
		{1024, u.Profile.Image1024},
		{10000, u.Profile.ImageOriginal},
	}
	bestDiff := 10000 * 3
	bestURL := ""
	for _, v := range fields {
		if v.Value == "" {
			continue
		}
		diff := v.Size - size
		if diff < 0 {
			diff = -diff
		}
		if diff == 0 {
			return v.Value
		}
		if diff < bestDiff {
			bestDiff = diff
			bestURL = v.Value
		}
	}
	return bestURL
}

type ChannelTopicPurpose struct {
	Value   string
	Creator UserID
	LastSet float64 `json:"last_set"`
}

type Channel struct {
	CacheTS  time.Time `json:"-"`
	NotExist bool      `json:"-"`

	ID         ChannelID
	Name       string
	IsChannel  bool        `json:"is_channel"`
	IsGroup    interface{} `json:"is_group"`
	IsMPIM     interface{} `json:"is_mpim"`
	Created    int         // unix millis
	Creator    UserID
	IsArchived bool `json:"is_archived"`
	IsGeneral  bool `json:"is_general"`
	HasPins    bool `json:"has_pins"`

	// The Members element may be out of date. The rtm.Client keeps an up-to-date list of
	// channel memberships for public and private channels. This can be used for MPIMs, however.
	Members    []UserID `json:"members"`
	NumMembers int      `json:"num_members"`

	// IM only
	IsUserDeleted bool `json:"is_user_deleted"`
	IsOpen        bool `json:"is_open"`

	Topic   ChannelTopicPurpose
	Purpose ChannelTopicPurpose
}

type ChannelIM struct {
	ID            ChannelID `json:"id"`
	User          UserID    `json:"user"`
	Created       int64     `json:"created"`
	IsUserDeleted bool      `json:"is_user_deleted"`
}

func (c *Channel) IsPublicChannel() bool {
	return c.IsChannel
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

type PinnedItem struct {
	Type      string    `json:"type"`
	Channel   ChannelID `json:"channel"`
	Created   int64     `json:"created"`
	CreatedBy UserID    `json:"created_id"`

	Message struct {
		TS        MessageTS `json:"ts"`
		Permalink string    `json:"permalink"`
	} `json:"message"`
	File struct {
		ID        FileID `json:"id"`
		Permalink string `json:"permalink"`
	} `json:"file"`
	Comment struct {
		ID      FileCommentID `json:"id"`
		Comment string
	} `json:"comment"`
}

type SelfPrefs struct {
	MutedChannels string `json:"muted_channels"`

	/*
		HighlightWords                     string      `json:"highlight_words"`
		UserColors                         string      `json:"user_colors"`
		ColorNamesInList                   bool        `json:"color_names_in_list"`
		GrowlsEnabled                      bool        `json:"growls_enabled"`
		Tz                                 string      `json:"tz"`
		PushDmAlert                        bool        `json:"push_dm_alert"`
		PushMentionAlert                   bool        `json:"push_mention_alert"`
		MsgReplies                         string      `json:"msg_replies"`
		PushEverything                     bool        `json:"push_everything"`
		PushShowPreview                    bool        `json:"push_show_preview"`
		PushIdleWait                       int         `json:"push_idle_wait"`
		PushSound                          string      `json:"push_sound"`
		PushLoudChannels                   string      `json:"push_loud_channels"`
		PushMentionChannels                string      `json:"push_mention_channels"`
		PushLoudChannelsSet                string      `json:"push_loud_channels_set"`
		EmailAlerts                        string      `json:"email_alerts"`
		EmailAlertsSleepUntil              int         `json:"email_alerts_sleep_until"`
		EmailMisc                          bool        `json:"email_misc"`
		EmailWeekly                        bool        `json:"email_weekly"`
		WelcomeMessageHidden               bool        `json:"welcome_message_hidden"`
		AllChannelsLoud                    bool        `json:"all_channels_loud"`
		LoudChannels                       string      `json:"loud_channels"`
		NeverChannels                      string      `json:"never_channels"`
		LoudChannelsSet                    string      `json:"loud_channels_set"`
		SearchSort                         string      `json:"search_sort"`
		ExpandInlineImgs                   bool        `json:"expand_inline_imgs"`
		ExpandInternalInlineImgs           bool        `json:"expand_internal_inline_imgs"`
		ExpandSnippets                     bool        `json:"expand_snippets"`
		PostsFormattingGuide               bool        `json:"posts_formatting_guide"`
		SeenWelcome2                       bool        `json:"seen_welcome_2"`
		SeenSsbPrompt                      bool        `json:"seen_ssb_prompt"`
		SpacesNewXpBannerDismissed         bool        `json:"spaces_new_xp_banner_dismissed"`
		SearchOnlyMyChannels               bool        `json:"search_only_my_channels"`
		SearchOnlyCurrentTeam              bool        `json:"search_only_current_team"`
		EmojiMode                          string      `json:"emoji_mode"`
		EmojiUse                           string      `json:"emoji_use"`
		HasInvited                         bool        `json:"has_invited"`
		HasUploaded                        bool        `json:"has_uploaded"`
		HasCreatedChannel                  bool        `json:"has_created_channel"`
		HasSearched                        bool        `json:"has_searched"`
		SearchExcludeChannels              string      `json:"search_exclude_channels"`
		MessagesTheme                      string      `json:"messages_theme"`
		WebappSpellcheck                   bool        `json:"webapp_spellcheck"`
		NoJoinedOverlays                   bool        `json:"no_joined_overlays"`
		NoCreatedOverlays                  bool        `json:"no_created_overlays"`
		DropboxEnabled                     bool        `json:"dropbox_enabled"`
		SeenDomainInviteReminder           bool        `json:"seen_domain_invite_reminder"`
		SeenMemberInviteReminder           bool        `json:"seen_member_invite_reminder"`
		MuteSounds                         bool        `json:"mute_sounds"`
		ArrowHistory                       bool        `json:"arrow_history"`
		TabUIReturnSelects                 bool        `json:"tab_ui_return_selects"`
		ObeyInlineImgLimit                 bool        `json:"obey_inline_img_limit"`
		NewMsgSnd                          string      `json:"new_msg_snd"`
		RequireAt                          bool        `json:"require_at"`
		SsbSpaceWindow                     string      `json:"ssb_space_window"`
		MacSsbBounce                       string      `json:"mac_ssb_bounce"`
		MacSsbBullet                       bool        `json:"mac_ssb_bullet"`
		ExpandNonMediaAttachments          bool        `json:"expand_non_media_attachments"`
		ShowTyping                         bool        `json:"show_typing"`
		PagekeysHandled                    bool        `json:"pagekeys_handled"`
		LastSnippetType                    string      `json:"last_snippet_type"`
		DisplayRealNamesOverride           int         `json:"display_real_names_override"`
		DisplayPreferredNames              bool        `json:"display_preferred_names"`
		Time24                             bool        `json:"time24"`
		EnterIsSpecialInTbt                bool        `json:"enter_is_special_in_tbt"`
		GraphicEmoticons                   bool        `json:"graphic_emoticons"`
		ConvertEmoticons                   bool        `json:"convert_emoticons"`
		SsEmojis                           bool        `json:"ss_emojis"`
		SidebarBehavior                    string      `json:"sidebar_behavior"`
		SeenOnboardingStart                bool        `json:"seen_onboarding_start"`
		OnboardingCancelled                bool        `json:"onboarding_cancelled"`
		SeenOnboardingSlackbotConversation bool        `json:"seen_onboarding_slackbot_conversation"`
		SeenOnboardingChannels             bool        `json:"seen_onboarding_channels"`
		SeenOnboardingDirectMessages       bool        `json:"seen_onboarding_direct_messages"`
		SeenOnboardingInvites              bool        `json:"seen_onboarding_invites"`
		SeenOnboardingSearch               bool        `json:"seen_onboarding_search"`
		SeenOnboardingRecentMentions       bool        `json:"seen_onboarding_recent_mentions"`
		SeenOnboardingStarredItems         bool        `json:"seen_onboarding_starred_items"`
		SeenOnboardingPrivateGroups        bool        `json:"seen_onboarding_private_groups"`
		OnboardingSlackbotConversationStep int         `json:"onboarding_slackbot_conversation_step"`
		DndEnabled                         bool        `json:"dnd_enabled"`
		DndStartHour                       string      `json:"dnd_start_hour"`
		DndEndHour                         string      `json:"dnd_end_hour"`
		MarkMsgsReadImmediately            bool        `json:"mark_msgs_read_immediately"`
		StartScrollAtOldest                bool        `json:"start_scroll_at_oldest"`
		SnippetEditorWrapLongLines         bool        `json:"snippet_editor_wrap_long_lines"`
		LsDisabled                         bool        `json:"ls_disabled"`
		SidebarTheme                       string      `json:"sidebar_theme"`
		SidebarThemeCustomValues           string      `json:"sidebar_theme_custom_values"`
		FKeySearch                         bool        `json:"f_key_search"`
		KKeyOmnibox                        bool        `json:"k_key_omnibox"`
		SpeakGrowls                        bool        `json:"speak_growls"`
		MacSpeakVoice                      string      `json:"mac_speak_voice"`
		MacSpeakSpeed                      int         `json:"mac_speak_speed"`
		CommaKeyPrefs                      bool        `json:"comma_key_prefs"`
		AtChannelSuppressedChannels        string      `json:"at_channel_suppressed_channels"`
		PushAtChannelSuppressedChannels    string      `json:"push_at_channel_suppressed_channels"`
		PromptedForEmailDisabling          bool        `json:"prompted_for_email_disabling"`
		FullTextExtracts                   bool        `json:"full_text_extracts"`
		NoTextInNotifications              bool        `json:"no_text_in_notifications"`
		NoMacelectronBanner                bool        `json:"no_macelectron_banner"`
		NoMacssb1Banner                    bool        `json:"no_macssb1_banner"`
		NoMacssb2Banner                    bool        `json:"no_macssb2_banner"`
		NoWinssb1Banner                    bool        `json:"no_winssb1_banner"`
		NoInvitesWidgetInSidebar           bool        `json:"no_invites_widget_in_sidebar"`
		NoOmniboxInChannels                bool        `json:"no_omnibox_in_channels"`
		KKeyOmniboxAutoHideCount           int         `json:"k_key_omnibox_auto_hide_count"`
		HideUserGroupInfoPane              bool        `json:"hide_user_group_info_pane"`
		MentionsExcludeAtUserGroups        bool        `json:"mentions_exclude_at_user_groups"`
		PrivacyPolicySeen                  bool        `json:"privacy_policy_seen"`
		EnterpriseMigrationSeen            bool        `json:"enterprise_migration_seen"`
		SearchExcludeBots                  bool        `json:"search_exclude_bots"`
		LoadLato2                          bool        `json:"load_lato_2"`
		FullerTimestamps                   bool        `json:"fuller_timestamps"`
		LastSeenAtChannelWarning           int         `json:"last_seen_at_channel_warning"`
		FlexResizeWindow                   bool        `json:"flex_resize_window"`
		MsgPreview                         bool        `json:"msg_preview"`
		MsgPreviewPersistent               bool        `json:"msg_preview_persistent"`
		EmojiAutocompleteBig               bool        `json:"emoji_autocomplete_big"`
		WinssbRunFromTray                  bool        `json:"winssb_run_from_tray"`
		WinssbWindowFlashBehavior          string      `json:"winssb_window_flash_behavior"`
		TwoFactorAuthEnabled               bool        `json:"two_factor_auth_enabled"`
		TwoFactorType                      interface{} `json:"two_factor_type"`
		TwoFactorBackupType                interface{} `json:"two_factor_backup_type"`
		ClientLogsPri                      string      `json:"client_logs_pri"`
		EnhancedDebugging                  bool        `json:"enhanced_debugging"`
		FlannelLazyMembers                 bool        `json:"flannel_lazy_members"`
		FlannelServerPool                  string      `json:"flannel_server_pool"`
		MentionsExcludeAtChannels          bool        `json:"mentions_exclude_at_channels"`
		ConfirmClearAllUnreads             bool        `json:"confirm_clear_all_unreads"`
		ConfirmUserMarkedAway              bool        `json:"confirm_user_marked_away"`
		BoxEnabled                         bool        `json:"box_enabled"`
		SeenSingleEmojiMsg                 bool        `json:"seen_single_emoji_msg"`
		ConfirmShCallStart                 bool        `json:"confirm_sh_call_start"`
		PreferredSkinTone                  string      `json:"preferred_skin_tone"`
		ShowAllSkinTones                   bool        `json:"show_all_skin_tones"`
		SeparatePrivateChannels            bool        `json:"separate_private_channels"`
		WhatsNewRead                       int         `json:"whats_new_read"`
		Hotness                            bool        `json:"hotness"`
		FrecencyJumper                     string      `json:"frecency_jumper"`
		FrecencyEntJumper                  string      `json:"frecency_ent_jumper"`
		Jumbomoji                          bool        `json:"jumbomoji"`
		NoFlexInHistory                    bool        `json:"no_flex_in_history"`
		NewxpSeenLastMessage               string      `json:"newxp_seen_last_message"`
		AttachmentsWithBorders             bool        `json:"attachments_with_borders"`
		ShowMemoryInstrument               bool        `json:"show_memory_instrument"`
		EnableUnreadView                   bool        `json:"enable_unread_view"`
		SeenUnreadViewCoachmark            bool        `json:"seen_unread_view_coachmark"`
		SeenCallsVideoBetaCoachmark        bool        `json:"seen_calls_video_beta_coachmark"`
		MeasureCSSUsage                    bool        `json:"measure_css_usage"`
		SeenRepliesCoachmark               bool        `json:"seen_replies_coachmark"`
		AllUnreadsSortOrder                string      `json:"all_unreads_sort_order"`
		Locale                             string      `json:"locale"`
		GdriveAuthed                       bool        `json:"gdrive_authed"`
		GdriveEnabled                      bool        `json:"gdrive_enabled"`
		SeenGdriveCoachmark                bool        `json:"seen_gdrive_coachmark"`
		ChannelSort                        string      `json:"channel_sort"`
		OverloadedMessageEnabled           bool        `json:"overloaded_message_enabled"`
		A11YFontSize                       string      `json:"a11y_font_size"`
		A11YAnimations                     bool        `json:"a11y_animations"`
	*/
}
