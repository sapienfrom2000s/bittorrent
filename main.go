package main

import (
	"bittorrent/torrent"
	"fmt"
	"log"
)

func main() {
	// Create TorrentFile with the path to your test torrent
	tf := torrent.TorrentFile{
		Path: "torrent/test.torrent",
	}

	// Parse the torrent file and get all info
	tfi, err := tf.SetTorrentFileInfo()
	if err != nil {
		log.Fatalf("Error parsing torrent file: %v", err)
	}

	peerManager := &torrent.PeerManager{
		Infohash: tfi.InfoHash,
	}
	trackerManager := torrent.TrackerManager{
		Infohash: tfi.InfoHash,
		Pm:       peerManager,
		Trackers: tfi.Trackers,
	}

	trackerManager.AskForPeers()
	for _, i := range peerManager.Peers {
		fmt.Println(i.Ip)
	}

	// pm := torrent.PeerManager{}
	// pm.InitTrackers(tfi.HTTPTrackers)
	// pm.InitTrackers(tfi.UDPTrackers)

	// Print out the parsed information
	fmt.Println("=== Torrent File Information ===")
	fmt.Printf("InfoHash: %s\n", tfi.InfoHash)
	fmt.Printf("Mode: %s\n", tfi.Mode)
	fmt.Printf("Piece Length: %d bytes\n", tfi.PieceLength)
	fmt.Printf("Total Pieces: %d\n", tfi.TotalPieces)

	fmt.Println("\n=== HTTP Trackers ===")
	for i, tracker := range tfi.Trackers {
		fmt.Printf("%d. %s. %s\n", i+1, tracker.Kind, tracker.Url)
	}

	fmt.Println("\n=== Info Dictionary Keys ===")
	for key := range tfi.Info {
		fmt.Printf("- %s\n", key)
	}

	fmt.Println("\n✅ Torrent file parsed successfully!")
}
