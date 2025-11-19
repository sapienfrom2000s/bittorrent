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

	fmt.Println("=== Torrent File Information ===")
	fmt.Printf("InfoHash: %s\n", tfi.InfoHash)
	fmt.Printf("Mode: %s\n", tfi.Mode)
	fmt.Printf("File Length: %d bytes (%.2f MB)\n", tfi.FileLength, float64(tfi.FileLength)/(1024*1024))
	fmt.Printf("Piece Length: %d bytes\n", tfi.PieceLength)
	fmt.Printf("Total Pieces: %d\n", tfi.TotalPieces)

	fmt.Println("\n=== Trackers ===")
	for i, tracker := range tfi.Trackers {
		fmt.Printf("%d. [%s] %s\n", i+1, tracker.Kind, tracker.Url)
	}

	fmt.Println("\n=== Info Dictionary Keys ===")
	for key := range tfi.Info {
		fmt.Printf("- %s\n", key)
	}

	fmt.Println("\n Torrent file parsed successfully!\n")

	idlePeerBus := &torrent.IdlePeerBus{
		Peer: make(chan *torrent.Peer),
	}

	blockRequestBus := &torrent.BlockRequestBus{
		BlockRequest: make(chan *torrent.BlockRequest),
	}

	blockRequestResponseBus := &torrent.BlockRequestResponseBus{
		BlockResponse: make(chan *torrent.BlockResponse),
	}

	blockWrittenBus := &torrent.BlockWrittenBus{
		BlockWritten: make(chan *torrent.BlockWritten),
	}

	peerManager := &torrent.PeerManager{
		Infohash:                tfi.InfoHash,
		IdlePeerBus:             idlePeerBus,
		BlockRequestBus:         blockRequestBus,
		BlockRequestResponseBus: blockRequestResponseBus,
	}

	trackerManager := &torrent.TrackerManager{
		Infohash:    tfi.InfoHash,
		Pm:          peerManager,
		Trackers:    tfi.Trackers,
		TotalPieces: uint(tfi.TotalPieces),
	}

	fmt.Println("\n Starting download...")

	// Start tracker communication in background
	go trackerManager.AskForPeers()

	pieceManager := &torrent.PieceManager{
		PieceLength: uint(tfi.PieceLength),
		FileLength:  uint(tfi.FileLength),
		TotalPieces: uint(tfi.TotalPieces),
	}

	err = pieceManager.InitPieces()
	if err != nil {
		log.Fatalf("Failed to initialize pieces: %v", err)
	}

	diskManager := &torrent.DiskManager{
		TorrentFileInfo: &tfi,
		BlockWrittenBus: blockWrittenBus,
	}

	// Scaffold files on disk before downloading
	diskManager.ScaffoldFiles()

	torrentManager := &torrent.TorrentManager{
		TorrentFilePath:         "torrent/test.torrent",
		PeerManager:             peerManager,
		PieceManager:            pieceManager,
		BlockRequestBus:         blockRequestBus,
		BlockRequestResponseBus: blockRequestResponseBus,
		BlockWrittenBus:         blockWrittenBus,
		DiskManager:             diskManager,
	}

	// Start background workers
	go peerManager.FindIdlePeers()
	go peerManager.ReadBlockRequestBus()

	// Start the download
	_, err = torrentManager.Download()
	if err != nil {
		log.Fatalf("Download failed: %v", err)
	}
}
