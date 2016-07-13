//Package mcping facilitates the pinging of Minecraft servers using the 1.7+ protocol.
package mcping

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/jsonq"
	"golang.org/x/net/context"
)

const (
	// DEFAULT_TIMEOUT stores default ping timeout
	DEFAULT_TIMEOUT = 1500
)

// Ping pings the specified server with the default timeout.
func Ping(addr string) (PingResponse, error) {
	return PingTimeout(addr, DEFAULT_TIMEOUT)
}

// PingTimeout pings the server with the provided timeout.
func PingTimeout(addr string, timeout int) (PingResponse, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()
	return PingContext(ctx, addr)
}

func PingWithTimeout(addr string, timeout time.Duration) (PingResponse, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return PingContext(ctx, addr)
}

// PingContext pings the server, returning early if the context expires.
func PingContext(ctx context.Context, addr string) (PingResponse, error) {
	type respOrError struct {
		r PingResponse
		e error
	}
	ch := make(chan respOrError)
	go func() {
		r, e := ping(ctx, addr)
		ch <- respOrError{r: r, e: e}
	}()

	select {
	case re := <-ch:
		return re.r, re.e
	case <-ctx.Done():
		return PingResponse{}, ErrTimeout{inner: ctx.Err()}
	}
}

func ping(ctx context.Context, addr string) (PingResponse, error) {
	var host string
	var port uint16
	var resp PingResponse

	//Start timer
	timer := pingTimer{}
	timer.Start()

	//Connect

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(time.Millisecond * DEFAULT_TIMEOUT)
	}
	conn, err := net.DialTimeout("tcp", addr, deadline.Sub(time.Now())/2)
	if err != nil {
		return resp, ErrConnect{err}
	}
	defer conn.Close()

	// If read/write (on a slow network?) takes longer than expected, abort
	conn.SetDeadline(deadline)
	connReader := bufio.NewReader(conn)

	var dataBuf bytes.Buffer

	var finBuf bytes.Buffer

	dataBuf.Write([]byte("\x00")) //Packet ID
	dataBuf.Write([]byte("\x04")) //Protocol Version 47

	if addrTokens := strings.Split(addr, ":"); len(addrTokens) == 2 {
		host = addrTokens[0]
		if intport, err := strconv.Atoi(addrTokens[1]); err == nil {
			port = uint16(intport)
		} else {
			return resp, ErrAddress(addr)
		}
	} else {
		return resp, ErrAddress(addr)
	}

	//Write host string length + host
	hostLength := uint8(len(host))
	dataBuf.Write([]uint8{hostLength})
	dataBuf.Write([]byte(host))

	//Write port
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, port)
	dataBuf.Write(b)

	//Next state ping
	dataBuf.Write([]byte("\x01"))

	//Prepend packet length with data
	packetLength := []byte{uint8(dataBuf.Len())}
	finBuf.Write(append(packetLength, dataBuf.Bytes()...))

	conn.Write(finBuf.Bytes())     //Sending handshake
	conn.Write([]byte("\x01\x00")) //Status ping

	//Get situationally useless full byte length
	binary.ReadUvarint(connReader)

	//Packet type 0 means we're good to receive ping
	packetType, _ := connReader.ReadByte()
	if bytes.Compare([]byte{packetType}, []byte("\x00")) != 0 {
		return resp, ErrPacketType(packetType)
	}

	//Get data length via Varint
	length, err := binary.ReadUvarint(connReader)
	if err != nil {
		return resp, ErrVarint{err}
	}
	if length < 10 {
		return resp, ErrSmallPacket(length)
	} else if length > 700000 {
		return resp, ErrBigPacket(length)
	}

	//Recieve json buffer
	bytesRecieved := uint64(0)
	recBytes := make([]byte, length)
	for bytesRecieved < length {
		n, _ := connReader.Read(recBytes[bytesRecieved:length])
		bytesRecieved = bytesRecieved + uint64(n)
	}

	//Stop Timer, collect latency
	latency := timer.End()

	pingString := string(recBytes)

	//Convert buffer into jsonq instance
	pingData := map[string]interface{}{}
	dec := json.NewDecoder(strings.NewReader(pingString))
	dec.Decode(&pingData)
	jq := jsonq.NewQuery(pingData)

	//Assemble PlayerSample
	playerSampleMap, err := jq.ArrayOfObjects("players", "sample")
	playerSamples := []PlayerSample{}
	for k := range playerSampleMap {
		sample := PlayerSample{}
		sample.UUID = playerSampleMap[k]["id"].(string)
		sample.Name = playerSampleMap[k]["name"].(string)
		playerSamples = append(playerSamples, sample)
	}

	//Assemble PingResponse
	resp.Latency = uint(latency)
	resp.Online, _ = jq.Int("players", "online")
	resp.Max, _ = jq.Int("players", "max")
	resp.Protocol, _ = jq.Int("version", "protocol")

	favicon, _ := jq.String("favicon")
	resp.Favicon = []byte(favicon)

	resp.Motd, _ = jq.String("description")
	versionStr, _ := jq.String("version", "name")
	arr := strings.Split(versionStr, " ")
	if len(arr) == 0 {
		resp.Server = "Unknown"
		resp.Version = "Unknown"
	} else if len(arr) == 1 {
		resp.Server = "Unknown"
		resp.Version = arr[0]
	} else if len(arr) == 2 {
		resp.Server = arr[0]
		resp.Version = arr[1]
	}
	resp.Sample = playerSamples

	return resp, nil
}
