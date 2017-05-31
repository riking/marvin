package rtm

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/util"
	"golang.org/x/net/websocket"
)

const MsgTypeAll = "_all"

const pingOnIdleTime = 5*time.Minute
const reconnectOnIdleTime = pingOnIdleTime + 15*time.Second

type SlackCodec struct{}

func (codec SlackCodec) Unmarshal(data []byte, payloadType byte, v interface{}) error {
	var msg *slack.RTMRawMessage

	msg = v.(*slack.RTMRawMessage)
	if payloadType == websocket.PongFrame {
		*msg = make(slack.RTMRawMessage)
		(*msg)["type"] = "pong"
		return nil
	}
	if payloadType != websocket.TextFrame {
		return errors.Errorf("Bad frame type, got %d", payloadType)
	}
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

	if c.conn == nil {

	}

	for {
		c.connLock.L.Lock()
		for {
			if c.conn == nil {
				c.reconnect()
				c.connLock.Wait()
				continue
			}
			break
		}
		conn := c.conn
		c.connLock.L.Unlock()

		msg = make(slack.RTMRawMessage)
		conn.SetReadDeadline(time.Now().Add(reconnectOnIdleTime))
		err = c.codec.Receive(conn, &msg)
		if err != nil {
			util.LogWarn("Websocket error calling wait:", err)
			c.connLock.L.Lock()
			c.reconnect()
			c.connLock.Wait()
			c.connLock.L.Unlock()
			continue
		}

		c.resetPingTimer()
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
		c.connLock.L.Lock()
		for {
			if c.conn == nil {
				c.reconnect()
				c.connLock.Wait()
				continue
			}
			w, err := c.conn.NewFrameWriter(websocket.TextFrame)
			if err != nil {
				util.LogWarn("Websocket write error:", err)
				c.reconnect()
				c.connLock.Wait()
				continue
			}
			w.Write(bytes)
			err = w.Close()
			if err != nil {
				util.LogWarn("Websocket write error:", err)
				c.reconnect()
				c.connLock.Wait()
				continue
			}
			break
		}
		c.connLock.L.Unlock()
	}
}

func (c *Client) pinger() {
	for range c.pingTimer.C {
		c.connLock.L.Lock()
		conn := c.conn
		c.connLock.L.Unlock()

		if conn == nil {
			// already reconnecting
			c.resetPingTimer()
			continue
		}

		msg := make(slack.RTMRawMessage)
		msg["type"] = "ping"
		msg["time"] = time.Now().Unix()
		c.SendMessageRaw(msg)
		util.LogGood("Pinged")
		continue
	}
}

func (c *Client) resetPingTimer() {
	c.pingTimer.Reset(pingOnIdleTime)
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
