package torrent

import "sync"

type Piece struct {
	index  uint
	length uint
}

type pieceData struct {
	peers  []Peer
	status string // downloaded, downloading, pending
	mu     sync.Mutex
}

type PieceManager struct {
	pending      []Piece
	downloaded   []Piece
	downloading  []Piece
	pieceDataMap map[uint]*pieceData // *pieceData to modify data directly
	mu           sync.Mutex
}
