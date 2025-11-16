package torrent

type PieceRequest struct {
	peer  Peer
	block []Block
	index uint
}

type IdlePeerBus struct {
	Peer chan *Peer
}

type PieceRequestBus struct {
	pieceRequest chan *PieceRequest
}

type PieceRequestResponseBus struct {
	data chan any
}

type TorrentManager struct {
	torrentFilePath string
	peerManager     *PeerManager
	pieceManager    *PieceManager
	// diskManager     *DiskManager
}

func (tm *TorrentManager) Download() (bool, error) {

	// event loop
	for {
		select {
			case
		}
	}

	return true, nil
}
