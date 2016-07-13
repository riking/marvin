//Package mcping facilitates the pinging of Minecraft servers using the 1.7+ protocol.
package mcping

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/jmoiron/jsonq"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	//DEFAULT_TIMEOUT stores default ping timeout
	DEFAULT_TIMEOUT = 1000
)

//Ping Pings with default timeout
func Ping(addr string) (PingResponse, error) {
	return ping(addr, DEFAULT_TIMEOUT)
}

//PingTimeout Pings with custom timeout
func PingTimeout(addr string, timeout int) (PingResponse, error) {
	return ping(addr, timeout)
}

func ping(addr string, timeout int) (PingResponse, error) {
	var host string
	var port uint16
	var resp PingResponse

	//Start timer
	timer := pingTimer{}
	timer.Start()

	//Connect
	conn, err := net.DialTimeout("tcp", addr, time.Millisecond*time.Duration(timeout))
	if err != nil {
		return resp, ErrConnect
	}
	defer conn.Close()


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
			return resp, err
		}
	} else {
		return resp, ErrAddress
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
		return resp, ErrPacketType
	}

	//Get data length via Varint
	length, err := binary.ReadUvarint(connReader)
	if err != nil {
		return resp, ErrVarint
	}
	if length < 10 {
		return resp, ErrSmallPacket
	} else if length > 700000 {
		return resp, ErrBigPacket
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
