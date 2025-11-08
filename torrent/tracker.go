package torrent

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"

	"github.com/jackpal/bencode-go"
)

type tracker struct {
	url  string
	kind string
}

func (t tracker) Peers(infoHash string) ([]any, error) {
	switch t.kind {
	case "http":
		return t.httpTrackerPeers(infoHash)
	case "udp":
		return nil, fmt.Errorf("udp not handled")
	}
	return nil, fmt.Errorf("only works for udp and http")
}

func (t tracker) httpTrackerPeers(infoHash string) ([]any, error) {
	params := url.Values{}
	params.Add("info_hash", infoHash)
	params.Add("peer_id", "abcde12345abcde12345")
	params.Add("port", "6883")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", "10000")
	params.Add("compact", "0")

	fullURL := t.url + "?" + params.Encode()
	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("Unable to get response from http tracker server")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http Server didn't throw 200")
	}

	dataMap, err := bencode.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Unable to decode response body")
	}

	dict, ok := dataMap.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("tracker response not in dictionary format")
	}

	peersMap, ok := dict["peers"].([]any)
	if !ok {
		return nil, fmt.Errorf("Peers map not in dictionary format")
	}

	return peersMap, nil
}

func (t tracker) ConnectRequest() (conn_id uint64, err error) {
	conn, err := net.Dial("udp", t.url)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	payload := make([]byte, 16)
	binary.BigEndian.PutUint64(payload[:8], 0x41727101980)
	binary.BigEndian.PutUint32(payload[8:12], 0)
	binary.BigEndian.PutUint32(payload[12:16], rand.Uint32())
	_, err = conn.Write(payload)
	if err != nil {
		return 0, err
	}

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	actionResponse := binary.BigEndian.Uint32(buffer[:4])
	if n != 16 {
		return 0, fmt.Errorf("Connect Response Size was supposed be 16 bytes, instead got %d bytes", n)
	}
	if actionResponse != 0 {
		return 0, fmt.Errorf("action Response was supposed to be 0, instead got %d", actionResponse)
	}
	conn_id = binary.BigEndian.Uint64(buffer[8:16])
	return conn_id, nil
}

// Offset  Size    Name    Value
// 0       64-bit integer  connection_id
// 8       32-bit integer  action          1 // announce
// 12      32-bit integer  transaction_id
// 16      20-byte string  info_hash
// 36      20-byte string  peer_id
// 56      64-bit integer  downloaded
// 64      64-bit integer  left
// 72      64-bit integer  uploaded
// 80      32-bit integer  event           0 // 0: none; 1: completed; 2: started; 3: stopped
// 84      32-bit integer  IP address      0 // default
// 88      32-bit integer  key
// 92      32-bit integer  num_want        -1 // default
// 96      16-bit integer  port
// 98

func (t tracker) AnnounceRequest(conn_id uint64, info_hash string) (p []any, err error) {
	payload := make([]byte, 20)
	binary.BigEndian.PutUint64(payload[:8], conn_id)
	binary.BigEndian.PutUint32(payload[8:12], 1)
	binary.BigEndian.PutUint32(payload[12:16], rand.Uint32())

	copy(payload[16:36], []byte(info_hash))
	peerID := "abcde12345abcde12345"
	copy(payload[36:56], []byte(peerID))
	binary.BigEndian.PutUint32(payload[56:64], 0)
	binary.BigEndian.PutUint32(payload[64:72], 10000)
	binary.BigEndian.PutUint32(payload[72:80], 0)
	binary.BigEndian.PutUint16(payload[80:84], 0)
	binary.BigEndian.PutUint16(payload[84:88], 0)
	key := rand.Uint32()
	binary.BigEndian.PutUint32(payload[88:92], key)
	num_want := -1
	binary.BigEndian.PutUint32(payload[92:96], uint32(int32(num_want)))
	binary.BigEndian.PutUint16(payload[96:98], 6888)

	conn, err := net.Dial("udp", t.url)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, err = conn.Write(payload)
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, 1024)
	_, err = conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	action := binary.BigEndian.Uint32(buffer[0:4])
	if action != 1 {
		return nil, fmt.Errorf("Announce response from udp tracker server had to be 1")
	}
}
