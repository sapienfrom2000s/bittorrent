package torrent

import (
	"fmt"
	"sync"
)

const blockLength = 16 * 1024

type Piece struct {
	status string // downloaded, downloading, pending
	index  uint
	length uint
	blocks []*Block
	mu     sync.Mutex
}

// Should block have a ref of Piece?
// The problem with that is that both will have ref of each other
// but generally data flows in only one direction
// Or am I thinking it in a wrong way as double linked list has ref
// of both ahead and behind block
type Block struct {
	status     string // downloaded, downloading, pending
	length     uint
	pieceIndex uint
	blockIndex uint
	offset     uint
	mu         sync.Mutex
}

type PieceManager struct {
	pending     map[int]*Piece
	downloaded  map[int]*Piece
	downloading map[int]*Piece
	pieces      map[int]*Piece
	pieceLength uint
	fileLength  uint
	totalPieces uint
	mu          sync.Mutex
}

func (pieceManager *PieceManager) InitPieces() error {
	if pieceManager.pieceLength == 0 || pieceManager.totalPieces == 0 {
		return fmt.Errorf("pieceLength or totalPieces is not initialized")
	}

	// Initialize maps
	pieceManager.pending = make(map[int]*Piece)
	pieceManager.downloaded = make(map[int]*Piece)
	pieceManager.downloading = make(map[int]*Piece)
	pieceManager.pieces = make(map[int]*Piece)

	for i := uint(1); i <= pieceManager.totalPieces; i++ {
		var pieceLength uint
		var lastPiece bool
		pieceLength = pieceManager.pieceLength
		if i == pieceManager.totalPieces {
			lastPiece = true
			pieceLength = pieceManager.fileLength - ((pieceManager.totalPieces - 1) * pieceManager.pieceLength)
		}

		piece := &Piece{
			status: "pending",
			index:  (i - uint(1)),
			length: pieceLength,
		}

		pieceIndex := int(i - uint(1))
		pieceManager.pending[pieceIndex] = piece
		pieceManager.pieces[pieceIndex] = piece

		pieceManager.initBlocks(piece, pieceLength, lastPiece)
	}
	return nil
}

// this piece length might be different from that of piece manager
func (pieceManager *PieceManager) initBlocks(piece *Piece, pieceLength uint, lastPiece bool) {
	numberOfBlocks := (pieceLength + blockLength - 1) / blockLength
	for i := uint(1); i <= numberOfBlocks; i++ {
		block := &Block{
			status:     "pending",
			length:     blockLength,
			pieceIndex: piece.index,
			blockIndex: (i - uint(1)),
			offset:     (i - uint(1)) * blockLength,
		}

		if lastPiece && (i == numberOfBlocks) {
			blockLen := pieceLength - ((numberOfBlocks - 1) * blockLength)
			block.length = blockLen
		}
		piece.blocks = append(piece.blocks, block)
	}
}

func (pieceManager *PieceManager) PendingPieces() map[int]*Piece {
	pieceManager.mu.Lock()
	defer pieceManager.mu.Unlock()

	return pieceManager.pending
}

// GetPiece returns a piece by its index from the pieces map
func (pieceManager *PieceManager) GetPiece(index int) *Piece {
	pieceManager.mu.Lock()
	defer pieceManager.mu.Unlock()

	return pieceManager.pieces[index]
}

// MovePieceToDownloaded moves a piece from pending to downloaded state
func (pieceManager *PieceManager) MovePieceToDownloaded(index int) error {
	pieceManager.mu.Lock()
	defer pieceManager.mu.Unlock()

	piece, exists := pieceManager.pending[index]
	if !exists {
		return fmt.Errorf("piece %d not found in pending", index)
	}

	delete(pieceManager.pending, index)
	piece.status = "downloaded"
	pieceManager.downloaded[index] = piece

	return nil
}
