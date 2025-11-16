package torrent

import (
	"fmt"
	"sync"
)

const blockLength = 16 * 1024

type Piece struct {
	index  uint
	length uint
	blocks []*Block
	mu     sync.Mutex
}

type Block struct {
	status string // downloaded, downloading, pending
	length uint
	index  uint
	offset uint
	mu     sync.Mutex
}

type PieceManager struct {
	pending     []*Piece
	downloaded  []*Piece
	downloading []*Piece
	pieces      []*Piece
	pieceLength uint
	fileLength  uint
	totalPieces uint
	mu          sync.Mutex
}

func (pm *PieceManager) InitPieces() error {
	if pm.pieceLength == 0 || pm.totalPieces == 0 {
		return fmt.Errorf("pieceLength or totalPieces is not initialized")
	}

	for i := uint(1); i <= pm.totalPieces; i++ {
		var pieceLength uint
		var lastPiece bool
		pieceLength = pm.pieceLength
		if i == pm.totalPieces {
			lastPiece = true
			pieceLength = pm.fileLength - ((pm.totalPieces - 1) * pm.pieceLength)
		}

		piece := &Piece{
			index:  (i - uint(1)),
			length: pieceLength,
		}

		pm.pending = append(pm.pending, piece)

		pm.initBlocks(piece, pieceLength, lastPiece)
	}
	return nil
}

// this piece length might be different from that of piece manager
func (pm *PieceManager) initBlocks(piece *Piece, pieceLength uint, lastPiece bool) {
	numberOfBlocks := (pieceLength + blockLength - 1) / blockLength
	for i := uint(1); i <= numberOfBlocks; i++ {
		block := &Block{
			status: "pending",
			length: blockLength,
			index:  (i - uint(1)),
			offset: (i - uint(1)) * blockLength,
		}

		if lastPiece && (i == numberOfBlocks) {
			blockLen := pieceLength - ((numberOfBlocks - 1) * blockLength)
			block.length = blockLen
		}
		piece.blocks = append(piece.blocks, block)
	}
}
