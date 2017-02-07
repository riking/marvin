package rss

import (
	"context"
	"fmt"
	"time"

	"github.com/riking/marvin/util"
)

type poller struct {
	mod *RSSModule
}

func (p *poller) init(mod *RSSModule) {
	p.mod = mod
}

func (p *poller) reportError(err error) {
	p.mod.team.SendMessage(p.mod.team.TeamConfig().LogChannel, fmt.Sprintf("[RSS Poller] Error: %+v", err))
}

func (p *poller) Run() {
	for {
		p.pollAll()
		util.LogGood("[RSS] poll complete")
		time.Sleep(1 * time.Hour)
	}
}

func (p *poller) pollAll() {
	util.LogGood("[RSS] beginning poll")
	feeds, err := p.mod.DB().GetAllSubscriptions()
	if err != nil {
		p.reportError(err)
	}
	for _, v := range feeds {
		ft := p.mod.GetFeedType(v.FeedType)
		if ft == nil {
			util.LogWarnf("[RSS] Unknown feed type %d (%c:%s)", ft, ft, v.FeedID)
			continue
		}
		_, err := p.pollFeed(ft, v.FeedID)
		if err != nil {
			util.LogBadf("[RSS] Error polling feed %c:%s\n%+v", ft, v.FeedID, err)
			continue
		}
	}
	return
}

func (p *poller) pollFeed(t FeedType, feedID string) (time.Duration, error) {
	// Load remote content
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	meta, items, err := t.LoadFeed(ctx, feedID)
	cancel()
	if err != nil {
		return 0, err
	}
	// Find new items
	itemIDs := make([]string, len(items))
	for i, v := range items {
		itemIDs[i] = v.ItemID()
	}
	itemIDs, err = p.mod.DB().GetUnseen(t.TypeID(), feedID, itemIDs)
	if err != nil {
		return 0, err
	}
	if len(itemIDs) == 0 {
		return 0, nil // all done
	}
	// Which channels to send to?
	channelList, err := p.mod.DB().GetFeedChannels(t.TypeID(), feedID)
	if err != nil {
		return 0, err
	}

	// Iterate backwards, so we move forwards in time
	for i := len(items) - 1; i >= 0; i-- {
		isNew := false
		id := items[i].ItemID()
		for _, v := range itemIDs {
			if v == id {
				isNew = true
			}
		}
		if !isNew {
			continue
		}

		slMessage := items[i].Render(meta)
		for _, ch := range channelList {
			go p.mod.team.SendComplexMessage(ch.Channel, slMessage)
		}
		p.mod.DB().MarkSeen(t.TypeID(), feedID, items[i].ItemID())
	}
	return meta.CacheAge(), nil
}
