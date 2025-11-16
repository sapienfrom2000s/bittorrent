package torrent

import (
	"fmt"
	"sync"
	"time"
)

type IdlePeerBus struct {
	Peer chan *Peer
}

// I think channel can be directly ref(IdlePeerBus) here.
// No need of decoupling this. Feels unnecessary
type PeerManager struct {
	Peers           []*Peer
	Infohash        string
	IdlePeerBus     *IdlePeerBus
	BlockRequestBus *BlockRequestBus
	mu              sync.Mutex
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

func (peerManager *PeerManager) ReadBlockRequestBus() {
	for {
		blockRequest := <-peerManager.BlockRequestBus.BlockRequest
		go peerManager.DownloadBlock(blockRequest)
	}
}

// Should this go inside peer instead of peerManager?
// The problem is that peer doesn't have ref to PeerManager
// which has BlockRequestResponseBus. So how will it respond back
// there?
func (PeerManager *PeerManager) DownloadBlock(blockrequest *BlockRequest) {
	peer := blockrequest.peer

	err := peer.DownloadBlock(blockrequest)
	if err != nil {
		fmt.Println(err)
	}
}
