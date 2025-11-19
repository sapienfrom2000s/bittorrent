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
	Status                  string // (idle/inactive/active)
	mu                      sync.Mutex
	PeerId                  string
	conn                    net.Conn
	BlockRequestResponseBus *BlockRequestResponseBus
	TotalPieces             uint // Total pieces in torrent (for bitfield initialization)
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

	// Read the peer's handshake response (68 bytes)
	response := make([]byte, 68)
	_, err = conn.Read(response)
	if err != nil {
		return fmt.Errorf("Failed to read handshake response: %v", err)
	}

	// Validate the handshake response
	if response[0] != 19 {
		return fmt.Errorf("Invalid handshake response")
	}

	fmt.Printf(" Handshake successful with peer %s\n", p.Ip)

	// Send interested message to peer
	err = p.sendInterested()
	if err != nil {
		return fmt.Errorf("Failed to send interested message: %v", err)
	}

	fmt.Printf(" Sent 'interested' message to peer %s\n", p.Ip)

	return nil
}

// sendInterested sends an "interested" message to the peer
// Message format: <len=0001><id=2>
func (p *Peer) sendInterested() error {
	message := make([]byte, 5)
	binary.BigEndian.PutUint32(message[0:4], 1) // length = 1
	message[4] = 2                              // message ID = 2 (interested)

	_, err := p.conn.Write(message)
	if err != nil {
		return err
	}

	p.am_interested = true
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

func (p *Peer) Listen() {
	conn := p.conn
	for {
		// Read message length prefix (4 bytes)
		lengthBuf := make([]byte, 4)
		_, err := conn.Read(lengthBuf)
		if err != nil {
			fmt.Printf("Peer %s disconnected: %v\n", p.Ip, err)
			return // Exit the goroutine on error
		}

		messageLength := binary.BigEndian.Uint32(lengthBuf)

		// Keep-alive message (length = 0)
		if messageLength == 0 {
			continue
		}

		// Read message ID (1 byte)
		messageIDBuf := make([]byte, 1)
		_, err = conn.Read(messageIDBuf)
		if err != nil {
			fmt.Printf("Failed to read message ID from %s: %v\n", p.Ip, err)
			return
		}
		messageID := messageIDBuf[0]

		// Read payload (remaining bytes)
		payloadLength := messageLength - 1

		payload := make([]byte, payloadLength)
		if payloadLength > 0 {
			bytesRead := 0
			for bytesRead < int(payloadLength) {
				n, err := conn.Read(payload[bytesRead:])
				if err != nil {
					fmt.Printf("Failed to read payload from %s: %v\n", p.Ip, err)
					return
				}
				bytesRead += n
			}
		}

		// Handle different message types
		switch messageID {
		case 0: // Peer choked me
			fmt.Printf(" Peer %s choked us\n", p.Ip)
			p.peerChokedMe()
		case 1: // peer unchoked me
			fmt.Printf(" Peer %s unchoked us - ready to download!\n", p.Ip)
			p.peerUnchokedMe()
		case 5: // peer sent bitfield
			fmt.Printf(" Received bitfield from peer %s (%d bytes)\n", p.Ip, len(payload))
			p.peerSentMeBitfield(payload)
			fmt.Printf(" DEBUG: Bitfield stored, length=%d, unchoked=%v\n", len(p.bitfield), p.unchoked)
		case 7: // Peer sent a piece(actually a block)
			fmt.Printf(" Received block data from peer %s (%d bytes)\n", p.Ip, len(payload))
			p.peerSentMeABlock(payload)
		default:
			// Ignore unknown message types
			fmt.Printf(" Unknown message type %d from peer %s\n", messageID, p.Ip)
		}
	}
}

func (p *Peer) peerChokedMe() {
	p.unchoked = false
}

func (p *Peer) peerUnchokedMe() {
	p.unchoked = true
	// Set to idle when unchoked (bitfield might have been sent earlier, or peer uses "have" messages)
	p.Status = "idle"
	if p.bitfield != nil && len(p.bitfield) > 0 {
		fmt.Printf(" Peer %s is now ready (has bitfield + unchoked)\n", p.Ip)
	} else {
		fmt.Printf(" Peer %s is now ready (unchoked, no bitfield yet - will use 'have' messages)\n", p.Ip)
	}
}

func (p *Peer) peerSentMeBitfield(payload []byte) {
	p.bitfield = payload
	// Set to idle if already unchoked
	if p.unchoked {
		p.Status = "idle"
		fmt.Printf(" Peer %s is now ready (has bitfield + unchoked)\n", p.Ip)
	}
}

// piece: <len=0009+X><id=7><index><begin><block>
// Payload format: 4 bytes piece index + 4 bytes begin offset + block data
func (p *Peer) peerSentMeABlock(payload []byte) {
	if len(payload) < 8 {
		fmt.Printf(" Invalid block payload size: %d bytes\n", len(payload))
		return
	}

	// Parse piece index (4 bytes)
	pieceIndex := binary.BigEndian.Uint32(payload[0:4])

	// Parse begin offset (4 bytes)
	begin := binary.BigEndian.Uint32(payload[4:8])

	// Calculate block index from begin offset
	blockIndex := begin / uint32(blockLength)

	// Extract block data (remaining bytes)
	blockData := payload[8:]

	fmt.Printf(" Parsed block: piece=%d, begin=%d, blockIndex=%d, dataSize=%d\n",
		pieceIndex, begin, blockIndex, len(blockData))

	blockResponse := &BlockResponse{
		pieceIndex: uint(pieceIndex),
		blockIndex: uint(blockIndex),
		blockData:  blockData,
	}

	// Set peer back to idle after receiving block
	p.Status = "idle"

	p.BlockRequestResponseBus.BlockResponse <- blockResponse
}

// request: <len=0013><id=6><index><begin><length>
// Message format: 4 bytes length prefix + 1 byte message ID + 4 bytes piece index + 4 bytes begin offset + 4 bytes block length
func (p *Peer) DownloadBlock(blockRequest *BlockRequest) error {
	block := blockRequest.block

	// Calculate the byte offset within the piece
	begin := uint32(block.blockIndex) * uint32(blockLength)

	// Create message: length prefix (4) + message ID (1) + index (4) + begin (4) + length (4) = 17 bytes
	message := make([]byte, 17)

	// Length prefix (13 bytes for the message after the length prefix)
	binary.BigEndian.PutUint32(message[0:4], 13)

	// Message ID (6 = request)
	message[4] = 6

	// Piece index (4 bytes)
	binary.BigEndian.PutUint32(message[5:9], uint32(block.pieceIndex))

	// Begin offset (4 bytes)
	binary.BigEndian.PutUint32(message[9:13], begin)

	// Block length (4 bytes)
	binary.BigEndian.PutUint32(message[13:17], uint32(blockLength))

	fmt.Printf(" Sending request: piece=%d, begin=%d, length=%d\n", block.pieceIndex, begin, blockLength)

	_, err := p.conn.Write(message)
	if err != nil {
		return err
	}
	return nil
}
