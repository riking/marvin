package autoinvite

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
)

const usageMass = "*`@marvin mass-invite`* will invite multiple people to a channel at once. " +
	"Arguments: a list of user mentions or usernames. " +
	"Use the command from the channel you want to invite users to."

func CmdMassInvite(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) == 0 {
		return marvin.CmdUsage(args, usageMass).WithSimpleUndo()
	}
	method := "channels.invite"
	if args.Source.ChannelID()[0] == 'G' {
		method = "groups.invite"
	} else if args.Source.ChannelID()[0] == 'C' {
		// ok
	} else {
		return marvin.CmdFailuref(args, "Cannot invite to a DM.").WithNoUndo()
	}

	var userIDs []slack.UserID
	for _, v := range args.Arguments {
		uid := t.ResolveUserName(v)
		if uid == "" {
			return marvin.CmdFailuref(args, "Error: '%s' is not a Slack username", v).WithSimpleUndo()
		}
		userIDs = append(userIDs, uid)
	}

	workers := 3
	if workers > len(userIDs)/2 {
		workers = (len(userIDs) / 2) + 1
	}
	var wg sync.WaitGroup
	var ch = make(chan slack.UserID)
	errCh := make(chan error, workers)
	counts := make([]int, workers)
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			var count = 0
			defer func() {
				counts[i] = count
			}()

			var form = make(url.Values)
			form.Set("channel", string(args.Source.ChannelID()))
			for uid := range ch {
				form.Set("user", string(uid))
				err := t.SlackAPIPostJSON(method, form, nil)
				if err != nil {
					errCh <- err
					return
				} else {
					count++
				}
			}
		}(i)
	}

	var firstErr error
	for _, v := range userIDs {
		select {
		case ch <- v:
			// ok
		case firstErr = <-errCh:
			break
		}
	}
	close(ch)
	wg.Wait()

	total := 0
	for i := range counts {
		total += counts[i]
	}
	if firstErr != nil {
		return marvin.CmdFailuref(args, "Error while inviting: %s\n%d users invited before the error.", firstErr, total)
	}

	return marvin.CmdSuccess(args, fmt.Sprintf("%d users invited.", total))
}
