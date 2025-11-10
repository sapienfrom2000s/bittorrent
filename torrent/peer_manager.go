package torrent

type PeerManager struct {
	peer []Peer
}

type Peer struct {
	infoHash      string
	id            string
	ip            string
	port          uint
	am_interested bool
	unchoked      bool
	bitfield      []bool
	status        string // (idle/inactive/active)
}
