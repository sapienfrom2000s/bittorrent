package torrent

import (
	"fmt"
	"sync"
)

type TrackerManager struct {
	Trackers    []tracker
	Infohash    string
	Pm          *PeerManager
	TotalPieces uint
	mu          sync.Mutex
}

func (tm *TrackerManager) AskForPeers() {
	// Stop after getting this many unique peers
	const maxPeers = 50

	for i, tracker := range tm.Trackers {
		// Check if we have enough peers
		currentPeerCount := len(tm.Pm.Peers)

		if currentPeerCount >= maxPeers {
			fmt.Printf("\n Got %d peers, skipping remaining %d trackers\n\n",
				currentPeerCount, len(tm.Trackers)-i)
			break
		}

		// There should be a timeout here
		peers, err := tracker.Peers(tm.Infohash)
		if err != nil {
			fmt.Printf(" %v\n", err)
			continue
		}

		newPeers := 0
		for _, peer := range peers {
			if tm.Pm.PeerExists(peer.Ip, peer.port) {
				continue
			}

			// Set TotalPieces for the peer
			peer.TotalPieces = tm.TotalPieces

			tm.Pm.InsertPeer(peer)
			newPeers++
			go tm.connectToPeer(peer)
		}

		if newPeers > 0 {
			totalPeers := len(tm.Pm.Peers)
			fmt.Printf("  Added %d new peers (total: %d)\n", newPeers, totalPeers)
		}
	}

	totalPeers := len(tm.Pm.Peers)
	fmt.Printf("\n Total unique peers: %d\n\n", totalPeers)
}

// connectToPeer establishes connection to a single peer
func (tm *TrackerManager) connectToPeer(peer *Peer) {
	err := peer.Handshake()
	if err != nil {
		// Silently fail - don't spam console with failed connections
		return
	}

	peer.Status = "connecting" // Will be set to "idle" after bitfield + unchoke
	go peer.Listen()
	fmt.Printf(" Connected to peer: %s\n", peer.Ip)
}
