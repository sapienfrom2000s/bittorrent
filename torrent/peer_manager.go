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
	Peers                   []*Peer
	Infohash                string
	IdlePeerBus             *IdlePeerBus
	BlockRequestBus         *BlockRequestBus
	BlockRequestResponseBus *BlockRequestResponseBus
	mu                      sync.Mutex
}

func (peerManager *PeerManager) PeerExists(ip string, port uint) bool {
	peerManager.mu.Lock()
	defer peerManager.mu.Unlock()
	for _, peer := range peerManager.Peers {
		if peer.Ip == ip && peer.port == port {
			return true
		}
	}
	return false
}

func (peerManager *PeerManager) InsertPeer(p *Peer) {
	peerManager.mu.Lock()
	defer peerManager.mu.Unlock()

	p.BlockRequestResponseBus = peerManager.BlockRequestResponseBus

	peerManager.Peers = append(peerManager.Peers, p)
}

// will run in a go routine
// will be touched only by a single routine
// so no lock needed
func (peerManager *PeerManager) FindIdlePeers() {
	fmt.Println(" Starting idle peer finder...")
	for {
		idleCount := 0
		for _, peer := range peerManager.Peers {
			if peer.Status == "idle" {
				peerManager.IdlePeerBus.Peer <- peer
				idleCount++
			}
		}
		if idleCount > 0 {
			fmt.Printf(" Found %d idle peer(s), sending to bus\n", idleCount)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (peerManager *PeerManager) ReadBlockRequestBus() {
	fmt.Println(" Starting block request bus reader...")
	for {
		blockRequest := <-peerManager.BlockRequestBus.BlockRequest
		fmt.Printf(" Processing block request (piece=%d, block=%d) for peer %s\n",
			blockRequest.block.pieceIndex, blockRequest.block.blockIndex, blockRequest.peer.Ip)
		go peerManager.DownloadBlock(blockRequest)
	}
}

// Should this go inside peer instead of peerManager?
// The problem is that peer doesn't have ref to PeerManager
// which has BlockRequestResponseBus. So how will it respond back
// there?
func (PeerManager *PeerManager) DownloadBlock(blockrequest *BlockRequest) {
	peer := blockrequest.peer

	// Mark peer as active while downloading
	peer.Status = "active"

	err := peer.DownloadBlock(blockrequest)
	if err != nil {
		fmt.Printf(" Failed to send block request: %v\n", err)
		// Set back to idle on error so it can be retried
		peer.Status = "idle"
	}
}
