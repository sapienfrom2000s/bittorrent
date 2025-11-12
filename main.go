package main

import (
	"bittorrent/torrent"
	"fmt"
)

func main() {
	file := torrent.TorrentFile{
		Path: "adsf",
	}
	torrentFileInfo, err := file.SetTorrentFileInfo()
	if err != nil {
		fmt.Println(err)
		return
	}
	// peerManager := torrent.PeerManager{}
	// pieceManager := torrent.PieceManager{}
	// torrentManager := torrent.TorrentManager{}
}
