package torrent

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackpal/bencode-go"
)

type tracker struct {
	Url  string
	Kind string
}

func (t tracker) Peers(infoHash string) ([]*Peer, error) {
	switch t.Kind {
	case "http":
		return t.httpTrackerPeers(infoHash)
	case "udp":
		return t.udpTrackerPeers(infoHash)
	}
	return nil, fmt.Errorf("only works for udp and http")
}

func (t tracker) httpTrackerPeers(infoHash string) ([]*Peer, error) {
	// Convert hex-encoded infohash to raw bytes
	infoHashBytes, err := hex.DecodeString(infoHash)
	if err != nil {
		return nil, fmt.Errorf("failed to decode infohash: %v", err)
	}

	params := url.Values{}
	// Use raw bytes for info_hash, not the hex string
	params.Add("info_hash", string(infoHashBytes))
	params.Add("peer_id", "abcde12345abcde12345")
	params.Add("port", "6883")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", "10000")
	params.Add("compact", "0")

	// Extract hostname for display
	parsedURL, _ := url.Parse(t.Url)
	hostname := parsedURL.Host
	if hostname == "" {
		hostname = t.Url
	}
	fmt.Printf(" HTTP tracker: %s", hostname)

	fullURL := t.Url + "?" + params.Encode()
	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("connection failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	dataMap, err := bencode.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("decode failed")
	}

	dict, ok := dataMap.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	// Check for failure reason first
	if failureReason, ok := dict["failure reason"].(string); ok {
		return nil, fmt.Errorf("%s", failureReason)
	}

	// Try to parse peers - could be compact (binary string) or dictionary format
	peers := make([]*Peer, 0)

	if peersCompact, ok := dict["peers"].(string); ok {
		peersBytes := []byte(peersCompact)
		if len(peersBytes)%6 != 0 {
			return nil, fmt.Errorf("invalid compact peers format: length %d not divisible by 6", len(peersBytes))
		}

		for i := 0; i < len(peersBytes); i += 6 {
			ip := fmt.Sprintf("%d.%d.%d.%d", peersBytes[i], peersBytes[i+1], peersBytes[i+2], peersBytes[i+3])
			port := binary.BigEndian.Uint16(peersBytes[i+4 : i+6])

			peer := Peer{
				id:       "", // Compact format doesn't include peer ID
				Ip:       ip,
				port:     uint(port),
				infoHash: string(infoHashBytes),
				PeerId:   "abcde12345abcde12345",
			}
			peers = append(peers, &peer)
		}
		fmt.Printf(" → %d peers\n", len(peers))
		return peers, nil
	}

	// Try dictionary format
	if peersMap, ok := dict["peers"].([]any); ok {
		for _, p := range peersMap {
			pDict, ok := p.(map[string]any)
			if !ok {
				continue
			}

			ip, _ := pDict["ip"].(string)
			port, _ := pDict["port"].(int64)
			peerID, _ := pDict["peer id"].(string)

			peer := Peer{
				id:   peerID,
				Ip:   ip,
				port: uint(port),
			}
			peers = append(peers, &peer)
		}
		fmt.Printf(" → %d peers\n", len(peers))
		return peers, nil
	}

	return nil, fmt.Errorf("no peers in response")
}

// udpTrackerPeers implements the full UDP tracker protocol
func (t tracker) udpTrackerPeers(infoHash string) ([]*Peer, error) {
	// Convert hex-encoded infohash to raw bytes
	infoHashBytes, err := hex.DecodeString(infoHash)
	if err != nil {
		return nil, fmt.Errorf("failed to decode infohash: %v", err)
	}

	// Parse UDP URL to extract host:port
	// Format: udp://hostname:port/announce
	trackerURL := t.Url

	// Remove "udp://" prefix
	if !strings.HasPrefix(trackerURL, "udp://") {
		return nil, fmt.Errorf("invalid UDP tracker URL: %s", trackerURL)
	}
	trackerURL = strings.TrimPrefix(trackerURL, "udp://")

	// Remove "/announce" suffix (or any path)
	if idx := strings.Index(trackerURL, "/"); idx != -1 {
		trackerURL = trackerURL[:idx]
	}

	fmt.Printf(" UDP tracker: %s", trackerURL)

	// Step 1: Connect to tracker and get connection ID
	conn, err := net.Dial("udp", trackerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to UDP tracker: %v", err)
	}
	defer conn.Close()

	// Set read timeout to avoid hanging forever
	conn.SetDeadline(time.Now().Add(15 * time.Second))

	connID, _, err := t.udpConnect(conn)
	if err != nil {
		return nil, err
	}

	// Step 2: Send announce request and get peers
	peers, err := t.udpAnnounce(conn, connID, infoHashBytes)
	if err != nil {
		return nil, err
	}

	fmt.Printf(" → %d peers\n", len(peers))
	return peers, nil
}

// udpConnect sends a connect request and returns the connection ID
func (t tracker) udpConnect(conn net.Conn) (uint64, uint32, error) {
	// Protocol ID for BitTorrent UDP tracker
	protocolID := uint64(0x41727101980)

	// Generate random transaction ID
	transactionID := rand.Uint32()

	// Build connect request: protocol_id (8) + action (4) + transaction_id (4) = 16 bytes
	request := make([]byte, 16)
	binary.BigEndian.PutUint64(request[0:8], protocolID)
	binary.BigEndian.PutUint32(request[8:12], 0) // action = 0 (connect)
	binary.BigEndian.PutUint32(request[12:16], transactionID)

	// Send connect request
	_, err := conn.Write(request)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to send connect request: %v", err)
	}

	// Read connect response: action (4) + transaction_id (4) + connection_id (8) = 16 bytes
	response := make([]byte, 16)
	n, err := conn.Read(response)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read connect response: %v", err)
	}
	if n != 16 {
		return 0, 0, fmt.Errorf("invalid connect response size: expected 16, got %d", n)
	}

	// Parse response
	action := binary.BigEndian.Uint32(response[0:4])
	respTransactionID := binary.BigEndian.Uint32(response[4:8])
	connectionID := binary.BigEndian.Uint64(response[8:16])

	// Validate response
	if action != 0 {
		return 0, 0, fmt.Errorf("invalid action in connect response: expected 0, got %d", action)
	}
	if respTransactionID != transactionID {
		return 0, 0, fmt.Errorf("transaction ID mismatch: expected %d, got %d", transactionID, respTransactionID)
	}

	return connectionID, transactionID, nil
}

// udpAnnounce sends an announce request and parses the peer list
// Announce request format (98 bytes):
// Offset  Size            Name            Value
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
func (t tracker) udpAnnounce(conn net.Conn, connectionID uint64, infoHashBytes []byte) ([]*Peer, error) {
	// Generate random transaction ID
	transactionID := rand.Uint32()

	// Build announce request (98 bytes)
	request := make([]byte, 98)
	binary.BigEndian.PutUint64(request[0:8], connectionID)
	binary.BigEndian.PutUint32(request[8:12], 1) // action = 1 (announce)
	binary.BigEndian.PutUint32(request[12:16], transactionID)

	// Info hash (20 bytes)
	copy(request[16:36], infoHashBytes)

	// Peer ID (20 bytes)
	peerID := "abcde12345abcde12345"
	copy(request[36:56], []byte(peerID))

	// Downloaded (8 bytes)
	binary.BigEndian.PutUint64(request[56:64], 0)

	// Left (8 bytes) - amount left to download
	binary.BigEndian.PutUint64(request[64:72], 10000000) // Arbitrary large number

	// Uploaded (8 bytes)
	binary.BigEndian.PutUint64(request[72:80], 0)

	// Event (4 bytes) - 0: none, 1: completed, 2: started, 3: stopped
	binary.BigEndian.PutUint32(request[80:84], 2) // 2 = started

	// IP address (4 bytes) - 0 = default
	binary.BigEndian.PutUint32(request[84:88], 0)

	// Key (4 bytes) - random
	binary.BigEndian.PutUint32(request[88:92], rand.Uint32())

	// Num_want (4 bytes) - -1 = default (0xFFFFFFFF in unsigned)
	binary.BigEndian.PutUint32(request[92:96], 0xFFFFFFFF)

	// Port (2 bytes)
	binary.BigEndian.PutUint16(request[96:98], 6881)

	// Send announce request
	_, err := conn.Write(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send announce request: %v", err)
	}

	// Read announce response
	// Minimum response size: action (4) + transaction_id (4) + interval (4) + leechers (4) + seeders (4) = 20 bytes
	// Plus 6 bytes per peer (4 IP + 2 port)
	response := make([]byte, 2048)
	n, err := conn.Read(response)
	if err != nil {
		return nil, fmt.Errorf("failed to read announce response: %v", err)
	}

	if n < 20 {
		return nil, fmt.Errorf("announce response too short: %d bytes", n)
	}

	// Parse response header
	action := binary.BigEndian.Uint32(response[0:4])
	respTransactionID := binary.BigEndian.Uint32(response[4:8])
	// interval := binary.BigEndian.Uint32(response[8:12])
	// leechers := binary.BigEndian.Uint32(response[12:16])
	// seeders := binary.BigEndian.Uint32(response[16:20])

	// Validate response
	if action != 1 {
		return nil, fmt.Errorf("invalid action in announce response: expected 1, got %d", action)
	}
	if respTransactionID != transactionID {
		return nil, fmt.Errorf("transaction ID mismatch: expected %d, got %d", transactionID, respTransactionID)
	}

	// Optionally log tracker stats (uncomment if needed)
	// fmt.Printf(" Tracker stats: interval=%ds, leechers=%d, seeders=%d\n", interval, leechers, seeders)

	// Parse peer list (starts at offset 20)
	peerData := response[20:n]
	if len(peerData)%6 != 0 {
		return nil, fmt.Errorf("invalid peer data length: %d (not divisible by 6)", len(peerData))
	}

	peers := make([]*Peer, 0)
	for i := 0; i < len(peerData); i += 6 {
		// IP address (4 bytes)
		ip := fmt.Sprintf("%d.%d.%d.%d", peerData[i], peerData[i+1], peerData[i+2], peerData[i+3])

		// Port (2 bytes)
		port := binary.BigEndian.Uint16(peerData[i+4 : i+6])

		peer := &Peer{
			id:       "",
			Ip:       ip,
			port:     uint(port),
			infoHash: string(infoHashBytes),
			PeerId:   peerID,
		}
		peers = append(peers, peer)
	}

	return peers, nil
}
