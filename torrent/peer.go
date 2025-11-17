package torrent

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
)

type Peer struct {
	id                      string
	Ip                      string
	port                    uint
	infoHash                string
	am_interested           bool
	unchoked                bool
	bitfield                []byte
	status                  string // (idle/inactive/active)
	mu                      sync.Mutex
	PeerId                  string
	conn                    net.Conn
	BlockRequestResponseBus *BlockRequestResponseBus
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

// Messages:
// Keep-Alive length = 0 (no ID) (no payload)
// Choke ID = 0 length = 1 payload: none
// Unchoke ID = 1 length = 1 payload: none
// Interested ID = 2 length = 1 payload: none
// NotInterested ID = 3 length = 1 payload: none
// Have ID = 4 length = 5 payload: piece index (4 bytes)
// Bitfield ID = 5 length = 1 + bitfield length payload: bitfield
// Request ID = 6 length = 13 payload: index (4), begin (4), length (4)
// Piece ID = 7 length = 9 + block size payload: index (4), begin (4),
// block
// Cancel ID = 8 length = 13 payload: index (4), begin (4), length (4)
//
// <length prefix><message ID><payload>

func (p *Peer) listen() {
	conn := p.conn
	for {
		buff := make([]byte, 16*1024)
		_, err := conn.Read(buff)
		if err != nil {
			fmt.Println("Unable to Read data")
		}
		var lengthPrefix uint
		binary.Decode(buff[0:1], binary.BigEndian, &lengthPrefix)

		var messageID uint
		binary.Decode(buff[1:2], binary.BigEndian, &messageID)

		switch messageID {
		case 0: // Peer choked me
			p.peerChokedMe()
		case 1: // peer unchoked me
			p.peerUnchokedMe()
		case 5: // peer sent bitfield
			p.peerSentMeBitfield(buff[2:])
		case 7: // Peer sent a piece(actually a block)
			p.peerSentMeABlock(buff[2:])
		}
	}
}

func (p *Peer) peerChokedMe() {
	p.unchoked = false
}

func (p *Peer) peerUnchokedMe() {
	p.unchoked = true
}

func (p *Peer) peerSentMeBitfield(payload []byte) {
	p.bitfield = payload
}

// piece: <len=0009+X><id=7><index><begin><block>
func (p *Peer) peerSentMeABlock(payload []byte) {
	pieceIndex := payload[0]
	blockIndex := payload[1]
	blockData := payload[2:]

	blockResponse := &BlockResponse{
		pieceIndex: uint(pieceIndex),
		blockIndex: uint(blockIndex),
		blockData:  blockData,
	}

	p.BlockRequestResponseBus.BlockResponse <- blockResponse
}

// request: <len=0013><id=6><index><begin><length>
func (p *Peer) DownloadBlock(blockRequest *BlockRequest) error {
	block := blockRequest.block
	payload := make([]byte, 6)

	payload[0] = byte(13)
	payload[1] = byte(6)
	payload[2] = byte(block.pieceIndex)
	payload[3] = byte(block.offset)
	binary.BigEndian.PutUint16(payload[4:6], blockLength)

	_, err := p.conn.Write(payload)
	if err != nil {
		return err
	}
	return nil
}
