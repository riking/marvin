package rss

import (
	"context"
	"fmt"
	"net/url"
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
		sqlMigrate2,
		sqlMigrate4,
	)
	t.DB().SyntaxCheck(
		sqlGetAllSubscriptions,
		sqlGetChannelSubscriptions,
		sqlGetFeedChannels,
		sqlCheckSeen,
		sqlMarkSeen,
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
	LoadFeed(ctx context.Context, feedID string) (FeedMeta, []Item, error)
}

// FeedMeta is a flag interface for information shared between multiple feed items.
type FeedMeta interface {
	FeedID() string
	CacheAge() time.Duration
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
	m, items, err := feedType.LoadFeed(loadCtx, feedID)
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
		mod.team.FormatChannel(args.Source.ChannelID()), feedType.Name(), feedID))
}
