package rss

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
)

func init() {
	marvin.RegisterModule(NewRSSModule)
}

const Identifier = "rss"

type RSSModule struct {
	team   marvin.Team
	db     *db
	poller *poller

	feedTypes map[string]FeedType
}

func NewRSSModule(t marvin.Team) marvin.Module {
	mod := &RSSModule{
		team: t,
		feedTypes: map[string]FeedType{
			"facebook": &FacebookType{},
			"twitter":  &TwitterType{},
		},
		db:     &db{t.DB()},
		poller: &poller{},
	}
	mod.poller.mod = mod
	return mod
}

func (mod *RSSModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *RSSModule) Load(t marvin.Team) {
	t.DB().MustMigrate(Identifier, 1486151238,
		sqlMigrate1,
		sqlMigrate2)
	t.DB().MustMigrate(Identifier, 1486452120,
		sqlMigrate3)

	t.DB().SyntaxCheck(
		sqlGetAllSubscriptions,
		sqlGetChannelSubscriptions,
		sqlGetFeedChannels,
		sqlCheckSeen,
		sqlMarkSeen,
		sqlLastSeen,
		sqlSubscribe,
		sqlUnsubscribe,
	)

	for _, v := range mod.feedTypes {
		v.OnLoad(mod)
	}
}

func (mod *RSSModule) Enable(t marvin.Team) {
	go mod.poller.Run()

	parent := marvin.NewParentCommand()
	subscribe := parent.RegisterCommandFunc("subscribe", mod.CommandSubscribe, usageSubscribe)
	parent.RegisterCommand("add", subscribe)
	parent.RegisterCommandFunc("list", mod.CommandList, usageList)
	unsub := parent.RegisterCommandFunc("unsubscribe", mod.CommandRemove, usageRemove)
	parent.RegisterCommand("remove", unsub)

	t.RegisterCommand("rss", parent)
}

func (mod *RSSModule) Disable(t marvin.Team) {
	t.OffAllEvents(Identifier)
}

func (mod *RSSModule) Config() marvin.ModuleConfig { return mod.team.ModuleConfig(Identifier) }
func (mod *RSSModule) DB() *db                     { return mod.db }

func (mod *RSSModule) GetFeedType(i TypeID) FeedType {
	for _, v := range mod.feedTypes {
		if v.TypeID() == i {
			return v
		}
	}
	return nil
}

func (mod *RSSModule) GetFeedTypeName(i TypeID) string {
	for _, v := range mod.feedTypes {
		if v.TypeID() == i {
			return v.Name()
		}
	}
	return fmt.Sprintf("%%!(Unrecognized type '%c')", i)
}

// ---

// FeedType is the basic polymorphism type of the RSS module.
type FeedType interface {
	// Return the type ID used in the database
	TypeID() TypeID
	// Preferred user-visible name of the feed type
	Name() string
	// OnLoad sets up the ModuleConfig keys.
	OnLoad(mod *RSSModule)
	// Domains gives a list of domains on which this module has priority.
	Domains() []string
	// VerifyFeedIdentifier parses a command input into a canonical feed ID,
	// and checks whether the feed in fact exists at all.
	// If config keys are missing, it returns ErrNotConfigured.
	VerifyFeedIdentifier(ctx context.Context, input string) (string, error)
	// LoadFeed pulls the current content of the given feed.
	LoadFeed(ctx context.Context, feedID string, lastSeen string) (FeedMeta, []Item, error)
}

// FeedMeta is a flag interface for information shared between multiple feed items.
type FeedMeta interface {
	FeedID() string
}

// An Item is something that has a per-feed item ID and can be rendered into a Slack message.
type Item interface {
	ItemID() string
	Render(FeedMeta) slack.OutgoingSlackMessage
}

var ErrNotConfigured = errors.New("Feed type not configured")

// ---

const usageSubscribe = `*COMMAND: ` + "`rss subscribe`" + `*
_SYNOPSIS_
  *@marvin rss* { *add* | *subscribe* } [ *facebook* | *rss* | *twitter* ] { _feed_url_ }
_DESCRIPTION_
  Add a feed to the channel. The most recent update, and any future new updates to the feed, will be posted to the channel.
  If a feed type (e.g. ` + "`facebook`" + `) is not specified, a guess will be made based on the URL.
  If a feed type is specified, sometimes you can use things other than a URL - for example, simply specifying the Twitter username is valid.`

func (mod *RSSModule) CommandSubscribe(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	// @marvin rss {add|subscribe} [facebook|rss|twitter] https://www.facebook.com/42Born2CodeUS
	// @marvin rss list
	// @marvin rss
	if mod.team.TeamConfig().IsReadOnly {
		return marvin.CmdFailuref(args, "Marvin is currently on read only.")
	}
	if len(args.Arguments) == 0 {
		return marvin.CmdUsage(args, usageSubscribe).WithSimpleUndo()
	}

	var feedType FeedType
	arg := args.Pop()
	if arg == "help" || len(arg) == 0 {
		return marvin.CmdUsage(args, usageSubscribe).WithSimpleUndo()
	} else if ft, ok := mod.feedTypes[arg]; ok {
		feedType = ft
		if len(args.Arguments) == 0 {
			return marvin.CmdFailuref(args, "Missing feed URL. Say `@marvin help rss subscribe` for help.").WithSimpleUndo()
		} else if len(args.Arguments) > 1 {
			return marvin.CmdFailuref(args, "Eh? Too many arguments! (`@marvin help rss subscribe` for help)").WithSimpleUndo()
		}
		arg = args.Pop()
		arg = slack.UnescapeText(arg)
	} else {
		arg = slack.UnescapeText(arg)
		uri, err := url.Parse(arg)
		if err != nil || uri.EscapedPath() == "" {
			return marvin.CmdFailuref(args, "'%s' does not appear to be a feed type or a URL.", arg).WithSimpleUndo()
		}
		for _, ft := range mod.feedTypes {
			found := false
			for _, v := range ft.Domains() {
				if uri.Host == v {
					found = true
					break
				}
			}
			if found {
				feedType = ft
				break
			}
		}
		// Default to RSS
		if feedType == nil {
			feedType = mod.feedTypes["rss"]
		}
	}

	if feedType == nil {
		return marvin.CmdFailuref(args, "Unknown feed type '%s'", arg).WithSimpleUndo()
	}

	feedID, err := feedType.VerifyFeedIdentifier(args.Ctx, arg)
	if err != nil {
		return marvin.CmdFailuref(args, "Could not verify that feed exists: %s", err).WithSimpleUndo()
	}
	loadCtx, cancel := context.WithTimeout(args.Ctx, 10*time.Second)
	m, items, err := feedType.LoadFeed(loadCtx, feedID, "")
	cancel()
	if err != nil {
		return marvin.CmdFailuref(args, "Could not perform initial feed query: %s", err).
			WithNoUndo()
	}
	var slackMessage slack.OutgoingSlackMessage
	if len(items) > 0 {
		slackMessage = items[0].Render(m)
	} else {
		slackMessage.Attachments = []slack.Attachment{{
			Color: "warning",
			Text:  ":warning: Feed appears to be empty! Subscribing anyway.",
		}}
	}

	if args.Source.ChannelID()[0] == 'D' {
		go mod.team.SendComplexMessage(args.Source.ChannelID(), slackMessage)
		return marvin.CmdSuccess(args, "The feed given validates; however, direct messages cannot be subscribed to RSS feeds.").WithSimpleUndo()
	} else {
		err = mod.DB().Subscribe(feedType.TypeID(), feedID, args.Source.ChannelID(), items)
		if err != nil {
			return marvin.CmdError(args, err, "Database error while subscribing.").WithNoUndo()
		}
	}

	go mod.team.SendComplexMessage(args.Source.ChannelID(), slackMessage)
	return marvin.CmdSuccess(args, fmt.Sprintf("%v is now subscribed to %s:%s",
		mod.team.FormatChannel(args.Source.ChannelID()), feedType.Name(), feedID)).WithNoUndo()
}

const usageList = `Use *@marvin rss list* [_channel_] to view the current RSS subscriptions and their IDs.
The ID is needed to remove or force-reload a subscription.`

func (mod *RSSModule) CommandList(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) != 0 && args.Arguments[0] == "help" {
		return marvin.CmdUsage(args, usageList).WithSimpleUndo()
	}

	targetChannel := args.Source.ChannelID()
	if len(args.Arguments) != 0 {
		specCh := t.ResolveChannelName(args.Arguments[0])
		if specCh != "" {
			targetChannel = specCh
		}
	}

	subs, err := mod.DB().GetChannelSubscriptions(targetChannel)
	if err != nil {
		return marvin.CmdError(args, err, "Database error")
	}

	var buf bytes.Buffer
	for i, v := range subs {
		fmt.Fprintf(&buf, "  %d: `%s:%s`\n", i+1, mod.GetFeedTypeName(v.FeedType), v.FeedID)
	}

	return marvin.CmdSuccess(args, fmt.Sprintf(
		"%d RSS subscriptions:\n"+
			"%s"+
			"Use the index (1, 2) or the identifier (e.g. `twitter:POTUS?with_replies`) as an argument to *`@marvin rss remove`* or *`@marvin rss refresh`*.",
		len(subs), buf.String())).WithSimpleUndo()
}

const usageRemove = `*COMMAND: ` + "`rss unsubscribe`" + `*
_SYNOPSIS_
  *@marvin rss* { *remove* | *unsubscribe* } [ _channel_ ] { _feed_number_ | _feed_id_ }
_DESCRIPTION_
  Remove a feed subscription from the channel.`

func (mod *RSSModule) CommandRemove(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) == 0 {
		return marvin.CmdUsage(args, usageRemove)
	}

	targetChannel := args.Source.ChannelID()

	arg := args.Pop()
	q := t.ResolveChannelName(arg)
	if q != "" {
		targetChannel = q
		if len(args.Arguments) == 0 {
			return marvin.CmdUsage(args, usageRemove)
		}
		arg = args.Pop()
	}

	var feedTypeID TypeID
	var feedID string
	if idx, err := strconv.ParseInt(arg, 10, 32); err == nil {
		idx := int(idx)
		subs, err := mod.DB().GetChannelSubscriptions(targetChannel)
		if err != nil {
			return marvin.CmdError(args, err, "Database error")
		}
		if idx < 0 {
			idx = len(subs) + idx + 1
		} else if idx == 0 {
			idx = 1
		}
		if idx < 0 || idx > len(subs) {
			return marvin.CmdFailuref(args, "Index '%s' out of range: Only %d rss subscriptions in %v",
				arg, len(subs), t.FormatChannel(targetChannel)).WithSimpleUndo()
		}
		feedID = subs[idx-1].FeedID
		feedTypeID = subs[idx-1].FeedType
	} else {
		colon := strings.IndexByte(arg, ':')
		if colon == -1 {
			return marvin.CmdFailuref(args, "Argument should be a number or a feed identifer from *@marvin rss list*").WithSimpleUndo()
		}
		feedTypeName := arg[:colon]
		for _, v := range mod.feedTypes {
			if v.Name() == feedTypeName {
				feedTypeID = v.TypeID()
			}
		}
		if feedTypeID == 0 {
			return marvin.CmdFailuref(args, "Unknown feed type '%s'", arg[:colon]).WithSimpleUndo()
		}
		feedID = arg[colon+1:]
	}

	found, err := mod.DB().Unsubscribe(feedTypeID, feedID, targetChannel)
	if err != nil {
		return marvin.CmdError(args, err, "Database error")
	}
	if !found {
		return marvin.CmdFailuref(args, "Feed subscription `%s:%s` not found",
			mod.GetFeedTypeName(feedTypeID), feedID).WithSimpleUndo()
	}
	return marvin.CmdSuccess(args, fmt.Sprintf("Subscription to `%s:%s` removed.",
		mod.GetFeedTypeName(feedTypeID), feedID)).WithNoUndo()
}
