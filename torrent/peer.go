package torrent

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
)

type Peer struct {
	id            string
	Ip            string
	port          uint
	infoHash      string
	am_interested bool
	unchoked      bool
	bitfield      []bool
	status        string // (idle/inactive/active)
	mu            sync.Mutex
	PeerId        string
	conn          net.Conn
}

// From unofficial docs <https://wiki.theory.org/BitTorrentSpecification:
// The handshake is a required message and must be the first message transmitted
// by the client. It is (49+len(pstr)) bytes long.
// handshake: <pstrlen><pstr><reserved><info_hash><peer_id>

func (p *Peer) Handshake() error {
	payload := make([]byte, 68)
	pstrlen := byte(uint8(19))
	payload[0] = pstrlen

	copy(payload[1:20], []byte("BitTorrent protocol"))

	binary.BigEndian.PutUint64(payload[20:28], 0)
	copy(payload[28:48], []byte(p.infoHash))
	copy(payload[48:68], []byte(p.PeerId))

	ipAddress := net.JoinHostPort(p.Ip, fmt.Sprintf("%d", p.port))
	conn, err := net.Dial("tcp", ipAddress)
	if err != nil {
		return fmt.Errorf("Failed to connect with Peer")
	}

	p.conn = conn

	_, err = conn.Write(payload)
	if err != nil {
		return fmt.Errorf("Failed to write to Peer")
	}

	// spawn a listen go routing to listen to peer's messages
	// go listen()

	return nil
}

// func (p *Peer) listen() {
// 	conn := p.conn
// }
