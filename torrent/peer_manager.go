package torrent

import (
	"sync"
	"time"
)

type PeerManager struct {
	Peers       []*Peer
	Infohash    string
	IdlePeerBus *IdlePeerBus
	mu          sync.Mutex
}

func (peerManager *PeerManager) PeerExists(id string) bool {
	peerManager.mu.Lock()
	defer peerManager.mu.Unlock()
	for _, peer := range peerManager.Peers {
		if peer.id == id {
			return true
		}
	}
	return false
}

func (peerManager *PeerManager) InsertPeer(p *Peer) {
	peerManager.mu.Lock()
	defer peerManager.mu.Unlock()

	peerManager.Peers = append(peerManager.Peers, p)
}

// will run in a go routine
// will be touched only by a single routine
// so no lock needed
func (peerManager *PeerManager) FindIdlePeers() {
	for {
		for _, peer := range peerManager.Peers {
			if peer.status == "idle" {
				peerManager.IdlePeerBus.Peer <- peer
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}
