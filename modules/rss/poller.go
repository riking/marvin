package rss

import (
	"context"
	"fmt"
	"time"
)

type poller struct {
	mod      *RSSModule
	onchange chan struct{}
	nextPoll map[TypeID]map[string]time.Time
}

func (p *poller) init(mod *RSSModule) {
	p.mod = mod
	p.onchange = make(chan struct{}, 1)
	p.nextPoll = make(map[TypeID]map[string]time.Time)
	p.onchange <- struct{}{}
}

func (p *poller) reportError(err error) {
	p.mod.team.SendMessage(p.mod.team.TeamConfig().LogChannel, fmt.Sprintf("[RSS Poller] Error: %+v", err))
}

func (p *poller) Run() {
	for {
		select {
		case <-p.onchange:
			p.rebuildNextPoll()
		default:
		}

	}
}

func (p *poller) SubscriptionsChanged() {
	select {
	case p.onchange <- struct{}{}:
	default:
		// already notified, will notice on next loop
	}
}

func (p *poller) getSleepTime() time.Duration {
	now := time.Now()
	shortestTime := time.Duration(24 * time.Hour)

	for _, m2 := range p.nextPoll {
		for _, v := range m2 {
			if v.IsZero() || v.Before(now) {
				shortestTime = 0
			} else if d := v.Sub(now); d < shortestTime {
				shortestTime = d
			}
		}
	}

	return shortestTime
}

func (p *poller) rebuildNextPoll() {
	subs, err := p.mod.DB().GetAllSubscriptions()
	if err != nil {
		p.reportError(err)
		return
	}
	missingMap := make(map[TypeID]map[string]bool)
	for t := range p.nextPoll {
		missingMap[t] = make(map[string]bool)
		for k := range p.nextPoll[t] {
			missingMap[t][k] = true
		}
	}
	for _, v := range subs {
		if p.nextPoll[v.FeedType] == nil {
			p.nextPoll[v.FeedType] = make(map[string]time.Time)
			p.nextPoll[v.FeedType][v.FeedID] = time.Time{}
		} else {
			delete(missingMap[v.FeedType], v.FeedID)
			_, ok := p.nextPoll[v.FeedType][v.FeedID]
			if !ok {
				p.nextPoll[v.FeedType][v.FeedID] = time.Time{}
			}
		}
	}
	for t := range missingMap {
		for k := range p.nextPoll[t] {
			delete(p.nextPoll[t], k)
		}
		if len(p.nextPoll[t]) == 0 {
			delete(p.nextPoll, t)
		}
	}
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
