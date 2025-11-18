package torrent

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
				blockRequest := &BlockRequest{
					block: block,
					peer:  peer,
				}

				tm.PeerManager.BlockRequestBus.BlockRequest <- blockRequest
			}
		case blockResponse := <-tm.PeerManager.BlockRequestResponseBus.BlockResponse:
			go tm.DiskManager.saveBlock(blockResponse)
		case blockWritten := <-tm.BlockWrittenBus.BlockWritten:
			go tm.handleBlockWritten(blockWritten)
		}
	}

	return true, nil
}

// Modern RAM bandwidth: ~20–50 GB/s
func (tm *TorrentManager) blockToBeRequested(peer *Peer) *Block {
	bitfield := peer.bitfield
	pendingPieces := tm.PieceManager.PendingPieces()

	var selectedBlock *Block
	for index, piece := range pendingPieces {
		// Check if peer has this piece in their bitfield
		if !(int(bitfield[index]) == 1) {
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
			// Piece completed successfully
			// Could add logging or progress tracking here
		}
	}
}
