TorrentManager (Orchestrator)
  ├── Polls PeerManager for idle peers
  ├── Polls PieceManager for pending pieces
  ├── Assigns work: DownloadPiece(peer, piece)
  │
  ├── PeerManager
  │   ├── Manages pool of connected peers
  │   ├── Tracks peer status (idle/downloading)
  │   ├── Handles peer communication via message buses
  │   └── Notifies when piece download completes
  │
  └── PieceManager
      ├── Tracks all pieces (pending/downloading/downloaded)
      ├── Maintains piece metadata (peers, frequency, status)
      ├── Returns pending pieces for download
      └── Updates piece status as downloads complete
