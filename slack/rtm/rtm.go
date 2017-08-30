package rtm

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/util"
	"golang.org/x/net/websocket"
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
	team marvin.Team

	connLock   *sync.Cond
	needReconn chan struct{}
	conn       *websocket.Conn
	codec      websocket.Codec
	pingTimer  *time.Timer

	membershipCh   chan membershipRequest
	channelMembers membershipMap

	MetadataLock sync.RWMutex
	Team struct {
		ID             slack.TeamID
		Name           string
		Domain         string
		EnterpriseID   slack.EnterpriseID `json:"enterprise_id"`
		EnterpriseName string `json:"enterprise_name"`
	}
	Self struct {
		ID   slack.UserID
		Name string
	}
	Users    []*slack.User
	Channels []*slack.Channel   // C
	Groups   []*slack.Channel   // G
	//Mpims    []*slack.Channel   // G
	Ims      []*slack.ChannelIM // D
	Bots []struct {
		ID      string `json:"id"`
		Deleted bool   `json:"deleted"`
		Name    string `json:"name"`
		Icons struct {
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
	Module       marvin.ModuleID
}

// Dial tries to connect to the Slack RTM API. The caller should register
// message handlers then call Start() to start the message pump.
func NewClient(team marvin.Team) *Client {
	c := &Client{}
	c.team = team

	var lock = new(sync.Mutex)
	c.connLock = sync.NewCond(lock)
	c.needReconn = make(chan struct{})
	c.pingTimer = time.NewTimer(0)

	cdc := SlackCodec{}
	c.codec = websocket.Codec{Marshal: cdc.Marshal, Unmarshal: cdc.Unmarshal}
	c.sendCbs = make(map[int]chan slack.RTMRawMessage)
	c.sendChan = make(chan []byte)

	c.channelMembers = make(membershipMap)
	c.membershipCh = make(chan membershipRequest, 8)

	go c.membershipWorker()
	go c.reconnectWorker()
	return c
}

func (c *Client) Connect() error {
	data := url.Values{}
	data.Set("token", c.team.TeamConfig().UserToken)
	data.Set("presence_sub", "true") // Only get user presence when requested
	var startResponse struct {
		URL            string
		CacheVersion   string `json:"cache_version"`
		CacheTsVersion string `json:"cache_ts_version"`
		Team struct {
			ID             slack.TeamID
			Name           string
			Domain         string
			EnterpriseID   slack.EnterpriseID `json:"enterprise_id"`
			EnterpriseName string `json:"enterprise_name"`
		}
		Self struct {
			ID   slack.UserID
			Name string
		}
	}
	err := c.team.SlackAPIPostJSON("rtm.start", data, &startResponse)
	if err != nil {
		return err
	}
	if startResponse.CacheTsVersion != "v2-bunny" {
		panic(errors.Errorf("Unexpected CacheTSVersion %s", startResponse.CacheTsVersion))
	}
	if startResponse.CacheVersion != "v16-giraffe" {
		panic(errors.Errorf("Unexpected CacheVersion %s", startResponse.CacheVersion))
	}
	wsURL, err := url.Parse(startResponse.URL)
	if err != nil {
		return errors.Wrap(err, "start RTM - could not parse URL")
	}
	originURL, err := url.Parse(fmt.Sprintf("https://%s.slack.com", c.team.Domain()))
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
		return errors.Wrap(err, "connect slack websocket")
	}

	c.MetadataLock.Lock()
	c.Self = startResponse.Self
	c.Team = startResponse.Team
	//c.Users = startResponse.Client.Users
	//c.AboutTeam = startResponse.Client.AboutTeam
	//c.Channels = startResponse.Client.Channels
	//c.Groups = startResponse.Client.Groups
	//c.Mpims = startResponse.Client.Mpims
	//c.Ims = startResponse.Client.Ims
	//c.Bots = startResponse.Client.Bots
	//c.LatestEventTs = startResponse.Client.LatestEventTs
	c.MetadataLock.Unlock()

	go c.getGroupList()

	var msg slack.RTMRawMessage
	err = c.codec.Receive(conn, &msg)
	if err != nil {
		return errors.Wrap(err, "receive first message from Slack")
	}
	if msg.Type() != "hello" {
		return errors.Errorf("Wrong type for first message, expected 'hello' got %s: %v", msg.Type(), msg)
	}

	c.connLock.L.Lock()
	c.conn = conn
	c.connLock.Broadcast()
	c.connLock.L.Unlock()

	c.dispatchMessage(msg)

	util.LogGood("Connected to Slack", startResponse.CacheVersion)
	return nil
}

func (c *Client) Start() {
	c.RegisterRawHandler("__internal", c.onChannelJoin, "channel_joined", nil)
	c.RegisterRawHandler("__internal", c.onGroupJoin, "group_joined", nil)
	c.RegisterRawHandler("__internal", c.onIMCreate, "im_created", nil)
	c.RegisterRawHandler("__internal", c.onTopicChange, "message", []string{"channel_topic", "group_topic"})
	c.RegisterRawHandler("__internal", c.onPurposeChange, "message", []string{"channel_purpose", "group_purpose"})

	c.RegisterRawHandler("__internal", c.onUserChange, "user_change", nil)
	c.RegisterRawHandler("__internal", c.onUserChange, "team_join", nil)

	c.RegisterRawHandler("__internal", c.onUserJoinChannel, "message", []string{"channel_join", "group_join"})
	c.RegisterRawHandler("__internal", c.onUserLeaveChannel, "message", []string{"channel_leave", "group_leave"})

	c.started = true
	go c.pump()
	go c.pumpSend()
	go c.pinger()
	c.reconnect()
}

func (c *Client) reconnect() {
	// called holding c.connLock
	select {
	case c.needReconn <- struct{}{}:
	default:
	}
}

func (c *Client) reconnectWorker() {
	doReconnect := func() {
		c.connLock.L.Lock()
		if c.conn != nil {
			c.conn.Close()
		}
		c.conn = nil
		c.connLock.L.Unlock()
		util.LogWarn("Disconnected.")

		for {
			util.LogWarn("Reconnecting...")
			err := c.Connect()
			if err != nil {
				util.LogBad("Could not reconnect", err)
				time.Sleep(30 * time.Second)
				continue
			}
			break
		}
		c.connLock.Broadcast()
	}

	for range c.needReconn {
		doReconnect()
	}
}

func (c *Client) RegisterRawHandler(
	mod marvin.ModuleID,
	cb func(slack.RTMRawMessage),
	typeOnly string, subtypes []string,
) {
	if typeOnly == "" && len(subtypes) > 0 {
		panic("cannot specify subtypes without specifying type")
	}

	c.msgCbsLock.Lock()
	defer c.msgCbsLock.Unlock()

	c.msgCbs = append(c.msgCbs, messageHandler{
		Cb:           cb,
		MsgType:      typeOnly,
		SubtypesOnly: subtypes,
	})
}

func (c *Client) UnregisterAllMatching(mod marvin.ModuleID) {
	c.msgCbsLock.Lock()
	defer c.msgCbsLock.Unlock()

	newMsgCbs := make([]messageHandler, 0, len(c.msgCbs))
	for _, v := range c.msgCbs {
		if v.Module != mod {
			newMsgCbs = append(newMsgCbs, v)
		}
	}
	c.msgCbs = newMsgCbs
	return
}

// SendMessage sends a simple message over the RTM api.
// When the Slack API returns an error, the error will be of type slack.CodedError.
func (c *Client) SendMessage(channelID slack.ChannelID, message string) (slack.RTMRawMessage, error) {
	if len(message) > 4000 {
		message = fmt.Sprintf("[TRUNCATED/MESSAGE TOO LONG]\n%s", message[:4100])
	}
	outgoing := make(slack.RTMRawMessage)
	outgoing["channel"] = string(channelID)
	outgoing["text"] = message
	return c.SendMessageRaw(outgoing)
}

func (c *Client) SendMessageRaw(rtmOut slack.RTMRawMessage) (slack.RTMRawMessage, error) {
	id := int32(c.rtmMsgId.Get())
	rtmOut["id"] = id
	if rtmOut["type"] == nil {
		rtmOut["type"] = "message"
	}
	bytes, err := json.Marshal(rtmOut)
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}
	respChan := make(chan slack.RTMRawMessage)
	c.sendCbsLock.Lock()
	c.sendCbs[int(id)] = respChan
	c.sendCbsLock.Unlock()
	c.sendChan <- bytes
	select {
	case respMsg := <-respChan:
		if rtmOut["type"] == "message" {
			fakeEvent := make(slack.RTMRawMessage)
			for k, v := range respMsg {
				if k != "reply_to" && k != "ok" && k != "_rawBytes" {
					fakeEvent[k] = v
				}
			}
			fakeEvent["team"] = string(c.Team.ID)
			fakeEvent["type"] = "message"
			fakeEvent["channel"] = rtmOut["channel"]
			fakeEvent["user"] = string(c.Self.ID)
			fakeEvent["_rawBytes"], _ = json.Marshal(fakeEvent)
			go c.dispatchMessage(fakeEvent)
		}

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
		util.LogBadf("[TIMEOUT] Reply to sent message %d timed out after 60 seconds", id)
		return nil, errors.Errorf("[TIMEOUT] Reply to %d timed out after 60 seconds", id)
	}
}
