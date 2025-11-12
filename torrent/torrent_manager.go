package torrent

type PieceRequest struct {
	peer  Peer
	index uint
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
}
