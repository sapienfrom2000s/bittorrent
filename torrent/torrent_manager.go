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

type BlockRequestBus struct {
	BlockRequest chan *BlockRequest
}

type BlockRequestResponseBus struct {
	BlockResponse chan *BlockResponse
}

type TorrentManager struct {
	TorrentFilePath         string
	PeerManager             *PeerManager
	PieceManager            *PieceManager
	BlockRequestBus         *BlockRequestBus
	BlockRequestResponseBus *BlockRequestResponseBus
	// diskManager     *DiskManager
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

		}
	}

	return true, nil
}

// Modern RAM bandwidth: ~20–50 GB/s
func (tm *TorrentManager) blockToBeRequested(peer *Peer) *Block {
	bitfield := peer.bitfield
	pendingPieces := tm.PieceManager.PendingPieces()

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
