package torrent

import "sync"

type PeerManager struct {
	Peers    []*Peer
	Infohash string
	mu       sync.Mutex
}

func (pm *PeerManager) PeerExists(id string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for _, peer := range pm.Peers {
		if peer.id == id {
			return true
		}
	}
	return false
}

func (pm *PeerManager) InsertPeer(p *Peer) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.Peers = append(pm.Peers, p)
}
