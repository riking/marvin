package rss

import (
	"context"
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
		poller: &poller{nextPoll: make(map[byte]map[string]time.Time)},
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
	)

	for _, v := range mod.feedTypes {
		v.OnLoad(mod)
	}
}

func (mod *RSSModule) Enable(t marvin.Team) {
}

func (mod *RSSModule) Disable(t marvin.Team) {
	t.OffAllEvents(Identifier)
}

func (mod *RSSModule) Config() marvin.ModuleConfig { return mod.team.ModuleConfig(Identifier) }
func (mod *RSSModule) DB() *db                     { return mod.db }

func (mod *RSSModule) Test() {

}

// ---

// FeedType is the basic polymorphism type of the RSS module.
type FeedType interface {
	// Return the type ID used in the database
	TypeID() TypeID
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
