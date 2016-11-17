package rtm

import (
	"encoding/json"

	"github.com/pkg/errors"
	"golang.org/x/net/websocket"

	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

const MsgTypeAll = "_all"

type SlackCodec struct{}

func (codec SlackCodec) Unmarshal(data []byte, payloadType byte, v interface{}) error {
	var msg *slack.RTMRawMessage

	if payloadType != websocket.TextFrame {
		return errors.Errorf("Bad frame type, got %d", payloadType)
	}
	msg = v.(*slack.RTMRawMessage)
	err := json.Unmarshal(data, msg)
	if err != nil {
		return errors.Wrap(err, "unmarshal json")
	}
	(*msg)[slack.MsgFieldRawBytes] = data
	return nil
}

func (codec SlackCodec) Marshal(v interface{}) (data []byte, payloadType byte, err error) {
	data, err = json.Marshal(v)
	if err != nil {
		return nil, 0, err
	}
	payloadType = websocket.TextFrame
	return data, payloadType, nil
}

func (c *Client) pump() {
	var msg slack.RTMRawMessage
	var err error

	msg = make(slack.RTMRawMessage)
	msg["type"] = "hello"
	c.dispatchMessage(msg)

	for {
		msg = make(slack.RTMRawMessage)
		err = c.codec.Receive(c.conn, &msg)
		if err != nil {
			panic(err) // TODO
		}

		if _, ok := msg["reply_to"]; ok {
			replyToId := msg.ReplyTo()
			c.sendCbsLock.Lock()
			ch, ok := c.sendCbs[replyToId]
			delete(c.sendCbs, replyToId)
			c.sendCbsLock.Unlock()
			if ok {
				ch <- msg
			}
		} else {
			c.dispatchMessage(msg)
		}
	}
}

func (c *Client) pumpSend() {
	for bytes := range c.sendChan {
		w, _ := c.conn.NewFrameWriter(websocket.TextFrame)
		w.Write(bytes)
		w.Close()
	}
}

func (c *Client) dispatchMessage(msg slack.RTMRawMessage) {
	c.msgCbsLock.RLock()
	defer c.msgCbsLock.RUnlock()

	for _, v := range c.msgCbs {
		if v.MsgType != MsgTypeAll && msg.Type() != v.MsgType {
			continue
		}
		if v.MsgType != MsgTypeAll && len(v.SubtypesOnly) != 0 {
			msgType := msg.Subtype()
			found := false
			for _, v := range v.SubtypesOnly {
				if msgType == v {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		go dispatchOne(v, msg)
	}
}

func dispatchOne(handler messageHandler, msg slack.RTMRawMessage) {
	defer func() {
		if err := recover(); err != nil {
			util.LogError(errors.Errorf("A message handler callback panicked: %+v", err))
		}
	}()

	handler.Cb(msg)
}
