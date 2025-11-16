package torrent

type BlockRequest struct {
	peer  *Peer
	block *Block
}

type BlockRequestBus struct {
	BlockRequest chan *BlockRequest
}

type BlockRequestResponseBus struct {
	Data chan any
}

type TorrentManager struct {
	torrentFilePath string
	peerManager     *PeerManager
	pieceManager    *PieceManager
	// diskManager     *DiskManager
}

func (tm *TorrentManager) Download() (bool, error) {

	// go routine to track Download
	// go routing to intercept Blocks from BlockRequestBus

	// event loop
	for {
		select {
		case peer := <-tm.peerManager.IdlePeerBus.Peer:
			block := tm.blockToBeRequested(peer)

			if block != nil {
				blockRequest := &BlockRequest{
					block: block,
					peer:  peer,
				}

				tm.peerManager.BlockRequestBus.BlockRequest <- blockRequest
			}
		case something:

			// ask peer to download a piece
			// case : listen for signal that download is done.
			// When it's done just exit out of it.
		}
	}

	return true, nil
}

// Modern RAM bandwidth: ~20–50 GB/s
func (tm *TorrentManager) blockToBeRequested(peer *Peer) *Block {
	bitfield := peer.bitfield
	pendingPieces := tm.pieceManager.PendingPieces()

	var selectedBlock *Block
	for _, piece := range pendingPieces {
		index := piece.index
		if !(int(bitfield[index]) == 1) {
			continue
		}

		selectedPiece := piece
		for _, pieceBlock := range selectedPiece.blocks {
			if pieceBlock.status == "pending" {
				selectedBlock = pieceBlock
			}
		}
	}
	return selectedBlock
}
