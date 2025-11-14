package torrent

import (
	"fmt"
	"sync"
)

type TrackerManager struct {
	Trackers []tracker
	Infohash string
	Pm       *PeerManager
	mu       sync.Mutex
}

func (tm *TrackerManager) AskForPeers() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, tracker := range tm.Trackers {
		peers, err := tracker.Peers(tm.Infohash)
		if err != nil {
			fmt.Println(err)
			continue
		}

		for _, peer := range peers {
			if tm.Pm.PeerExists(peer.id) {
				continue
			}

			tm.Pm.InsertPeer(peer)
		}
	}
}
