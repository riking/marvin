package rtm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/websocket"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
)

type uniqueID struct {
	counter int32
}

func (c *uniqueID) Get() int32 {
	for {
		val := atomic.LoadInt32(&c.counter)
		if atomic.CompareAndSwapInt32(&c.counter, val, val+1) {
			return val
		}
	}
}

type Client struct {
	conn  *websocket.Conn
	codec websocket.Codec
	team  marvin.Team

	reconnectURL string

	metadataLock sync.RWMutex
	Self         struct {
		ID             slack.UserID
		Name           string
		Prefs          slack.SelfPrefs
		Created        int
		ManualPresence string `json:"manual_presence"`
	}
	Users     []slack.User
	AboutTeam slack.TeamInfo  `json:"team"`
	Channels  []slack.Channel // C
	Groups    []slack.Channel // G
	Mpims     []slack.Channel // G
	Ims       []slack.Channel // D
	Bots      []struct {
		ID      string `json:"id"`
		Deleted bool   `json:"deleted"`
		Name    string `json:"name"`
		Icons   struct {
			Image36 string `json:"image_36"`
			Image48 string `json:"image_48"`
			Image72 string `json:"image_72"`
		} `json:"icons"`
	} `json:"bots"`
	LatestEventTs string `json:"latest_event_ts"`

	sendChan chan []byte

	msgCbsLock sync.RWMutex
	started    bool // if false, no need to lock
	msgCbs     []messageHandler

	sendCbsLock sync.Mutex
	sendCbs     map[int]chan slack.RTMRawMessage

	rtmMsgId uniqueID
}

type messageHandler struct {
	Cb           func(slack.RTMRawMessage)
	MsgType      string
	SubtypesOnly []string
	UnregisterID string
}

const startAPIURL = "https://slack.com/api/rtm.start"

// Dial tries to connect to the Slack RTM API. The caller should register
// message handlers then call Start() to start the message pump.
func Dial(team marvin.Team) (*Client, error) {
	data := url.Values{}
	data.Set("token", team.TeamConfig().UserToken)
	data.Set("no-unreads", "true")
	data.Set("mipm-aware", "true")
	var startResponse struct {
		*slack.APIResponse
		URL            string
		CacheVersion   string `json:"cache_version"`
		CacheTsVersion string `json:"cache_ts_version"`
		*Client
	}
	resp, err := team.SlackAPIPost(startAPIURL, data)
	if err != nil {
		return nil, errors.Wrap(err, "slack post rtm.start")
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "slack post rtm.start read body")
	}
	err = json.Unmarshal(respBytes, &startResponse)
	if err != nil {
		return nil, errors.Wrap(err, "slack post rtm.start unmarshal")
	}
	if !startResponse.OK {
		return nil, errors.Wrap(startResponse.APIResponse, "slack post rtm.start error")
	}
	wsURL, err := url.Parse(startResponse.URL)
	if err != nil {
		return nil, errors.Wrap(err, "slack post rtm.start - could not parse URL")
	}
	originURL, err := url.Parse(fmt.Sprintf("https://%s.slack.com", team.Domain()))
	if err != nil {
		panic(errors.Wrap(err, "could not parse URL of team domain"))
	}
	wsCfg := websocket.Config{
		Location: wsURL,
		Origin:   originURL,
		Version:  websocket.ProtocolVersionHybi,
	}
	conn, err := websocket.DialConfig(&wsCfg)
	if err != nil {
		return nil, errors.Wrap(err, "connect slack websocket")
	}
	c := startResponse.Client
	c.conn = conn
	c.team = team
	cdc := SlackCodec{}
	c.codec = websocket.Codec{cdc.Marshal, cdc.Unmarshal}
	c.sendCbs = make(map[int]chan slack.RTMRawMessage)

	var msg slack.RTMRawMessage
	err = c.codec.Receive(c.conn, &msg)
	if err != nil {
		return nil, errors.Wrap(err, "receive first message from Slack")
	}
	if msg.Type() != "hello" {
		return nil, errors.Errorf("Wrong type for first message, expected 'hello' got %s: %v", msg.Type(), msg)
	}
	c.sendChan = make(chan []byte)
	fmt.Println(c)
	return c, nil
}

func (c *Client) Start() {
	c.RegisterRawHandler("__internal", c.onChannelCreate, "channel_created", nil)

	c.started = true
	go c.pump()
	go c.pumpSend()
}

func (c *Client) RegisterRawHandler(
	unregisterID string,
	cb func(slack.RTMRawMessage),
	typeOnly string, subtypes []string,
) {
	if typeOnly == "" && len(subtypes) > 0 {
		panic("cannot specify subtypes without specifying type")
	}

	if c.started {
		c.msgCbsLock.Lock()
		defer c.msgCbsLock.Unlock()
	}

	c.msgCbs = append(c.msgCbs, messageHandler{
		Cb:           cb,
		MsgType:      typeOnly,
		SubtypesOnly: subtypes,
	})
}

func (c *Client) UnregisterAllMatching(unregisterID string) {
	c.msgCbsLock.Lock()
	defer c.msgCbsLock.Unlock()

	newMsgCbs := make([]messageHandler, 0, len(c.msgCbs))
	for _, v := range c.msgCbs {
		if v.UnregisterID != unregisterID {
			newMsgCbs = append(newMsgCbs, v)
		}
	}
	c.msgCbs = newMsgCbs
	return
}

// SendMessage sends a simple message over the RTM api.
// When the Slack API returns an error, the error will be of type slack.CodedError.
func (c *Client) SendMessage(channelID slack.ChannelID, message string) (slack.RTMRawMessage, error) {
	var msg struct {
		ID      int32  `json:"id"`
		Type    string `json:"type"`
		Channel string `json:"channel"`
		Text    string `json:"text"`
	}
	msg.ID = c.rtmMsgId.Get()
	msg.Type = "message"
	msg.Channel = string(channelID)
	msg.Text = message
	bytes, err := json.Marshal(msg)
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}
	respChan := make(chan slack.RTMRawMessage)
	c.sendCbsLock.Lock()
	c.sendCbs[int(msg.ID)] = respChan
	c.sendCbsLock.Unlock()
	c.sendChan <- bytes
	select {
	case respMsg := <-respChan:
		var resp struct {
			Ok    bool             `json:"ok"`
			Error slack.CodedError `json:"error"`
		}
		respMsg.ReMarshal(&resp)
		if resp.Ok {
			return respMsg, nil
		} else {
			return respMsg, resp.Error
		}
	case <-time.After(1 * time.Minute):
		return nil, errors.Errorf("[TIMEOUT] Reply to %d timed out after 60 seconds", msg.ID)
	}
}
