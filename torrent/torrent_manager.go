package torrent

import "fmt"

type BlockRequest struct {
	peer  *Peer
	block *Block
}

type BlockResponse struct {
	pieceIndex uint
	blockIndex uint
	blockData  []byte
}

type BlockWritten struct {
	pieceIndex uint
	blockIndex uint
	success    bool
	err        error
}

type BlockRequestBus struct {
	BlockRequest chan *BlockRequest
}

type BlockRequestResponseBus struct {
	BlockResponse chan *BlockResponse
}

type BlockWrittenBus struct {
	BlockWritten chan *BlockWritten
}

type TorrentManager struct {
	TorrentFilePath         string
	PeerManager             *PeerManager
	PieceManager            *PieceManager
	BlockRequestBus         *BlockRequestBus
	BlockRequestResponseBus *BlockRequestResponseBus
	BlockWrittenBus         *BlockWrittenBus
	DiskManager             *DiskManager
}

func (tm *TorrentManager) Download() (bool, error) {

	// go routine to track Download
	// go routing to intercept Blocks from BlockRequestBus

	// event loop
	for {
		select {
		case peer := <-tm.PeerManager.IdlePeerBus.Peer:
			block := tm.blockToBeRequested(peer)

			if block != nil {
				fmt.Printf(" Requesting block (piece=%d, block=%d) from peer %s\n",
					block.pieceIndex, block.blockIndex, peer.Ip)
				blockRequest := &BlockRequest{
					block: block,
					peer:  peer,
				}

				tm.PeerManager.BlockRequestBus.BlockRequest <- blockRequest
			} else {
				fmt.Printf(" No block to request from peer %s (bitfield empty or no pending pieces)\n", peer.Ip)
			}
		case blockResponse := <-tm.PeerManager.BlockRequestResponseBus.BlockResponse:
			fmt.Printf(" Received block response (piece=%d, block=%d) - sending to disk\n",
				blockResponse.pieceIndex, blockResponse.blockIndex)
			go tm.DiskManager.saveBlock(blockResponse)
		case blockWritten := <-tm.BlockWrittenBus.BlockWritten:
			if blockWritten.success {
				fmt.Printf(" Block written to disk (piece=%d, block=%d)\n",
					blockWritten.pieceIndex, blockWritten.blockIndex)
			} else {
				fmt.Printf(" Failed to write block (piece=%d, block=%d): %v\n",
					blockWritten.pieceIndex, blockWritten.blockIndex, blockWritten.err)
			}
			go tm.handleBlockWritten(blockWritten)
		}
	}

	return true, nil
}

// Modern RAM bandwidth: ~20–50 GB/s
func (tm *TorrentManager) blockToBeRequested(peer *Peer) *Block {
	bitfield := peer.bitfield

	// Check if peer has sent bitfield yet
	if bitfield == nil || len(bitfield) == 0 {
		fmt.Printf(" Peer %s has no bitfield yet\n", peer.Ip)
		return nil
	}

	pendingPieces := tm.PieceManager.PendingPieces()
	fmt.Printf(" Checking %d pending pieces against peer %s bitfield\n", len(pendingPieces), peer.Ip)

	var selectedBlock *Block
	for index, piece := range pendingPieces {
		// Check if peer has this piece in their bitfield
		// Bitfield is a byte array where each bit represents a piece
		byteIndex := index / 8
		bitIndex := uint(index % 8)

		// Check if we have enough bytes in bitfield
		if byteIndex >= len(bitfield) {
			continue
		}

		// Check if the bit is set (peer has this piece)
		hasPiece := (bitfield[byteIndex] & (1 << (7 - bitIndex))) != 0
		if !hasPiece {
			continue
		}

		// Find a pending block in this piece
		for _, pieceBlock := range piece.blocks {
			if pieceBlock.status == "pending" {
				selectedBlock = pieceBlock
				break
			}
		}

		// If we found a block, return it
		if selectedBlock != nil {
			break
		}
	}
	return selectedBlock
}

func (tm *TorrentManager) handleBlockWritten(event *BlockWritten) {
	piece := tm.PieceManager.GetPiece(int(event.pieceIndex))
	if piece == nil {
		return
	}

	// Update block status
	piece.mu.Lock()
	if int(event.blockIndex) < len(piece.blocks) {
		block := piece.blocks[event.blockIndex]
		block.mu.Lock()
		block.status = "downloaded"
		block.mu.Unlock()
	}

	// Check if all blocks in piece are downloaded
	allDownloaded := true
	for _, b := range piece.blocks {
		b.mu.Lock()
		if b.status != "downloaded" {
			allDownloaded = false
		}
		b.mu.Unlock()
		if !allDownloaded {
			break
		}
	}
	piece.mu.Unlock()

	// If all blocks downloaded, move piece to downloaded state
	if allDownloaded {
		err := tm.PieceManager.MovePieceToDownloaded(int(event.pieceIndex))
		if err == nil {
			fmt.Printf(" PIECE %d COMPLETED! Moving to downloaded state\n", event.pieceIndex)

			// Calculate and display progress
			downloaded := len(tm.PieceManager.Downloaded())
			total := int(tm.PieceManager.TotalPieces)
			percentage := float64(downloaded) / float64(total) * 100
			fmt.Printf(" Progress: %d/%d pieces (%.2f%%)\n", downloaded, total, percentage)
		}
	}
}
