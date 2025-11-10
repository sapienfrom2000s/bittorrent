System Design for bittorrent client

Torrent Manager
Peer Manager
Piece Manager
Disk Manager

type Peer {
  infoHash string
  id string
  ip string
  port int
  am_interested bool
  unchoked bool
  bitfield []bool
  status string(blocked/inactive/active/downloading)
}

type struct TorrentManager {
  torrentFile TorrentFile
  peers []Peer
}

Torrent Manager
  Init
    1. Parse the torrent file
    2. Ask for peers from tracker
    3. Get peers
    4. Ask Peer Manager to init Peers(handshake and interested)
    5. Init Piece Manager
    6. Init Disk Manager
  Start Event Loop
    1. Start a go routine whose job is to find a idle active peer. Get a piece from piece manager
       and ask the torrent manager to download it.(Ask for a Piece Manager)
    2. Start a event loop to handle incoming messages from Peer Manager, Ask for a Piece Manager, Trigger Piece Request to
       Peer Manager.
    3. When you receive piece from torrent manager, ask disk manager to save it.




### Communication bw Torrent Manager, Peer Manager and Peer

```go
type PieceRequestMessageBus struct {
  data chan any
}

type PieceRequestResponseMessageBus struct {
  data chan any
}

// instance to create message buses for both request and response
pieceRequestMessageBus := PieceRequestMessageBus()
pieceResponseMessageBus := PieceRequestResponseMessageBus()

type struct TorrentManager {
  torrentFilePath TorrentFilePath
  peers []Peer
  pieceRequestMesssageBus *PieceRequestMessageBus
  pieceRequestResponseMessageBus *PieceRequestResponseMessageBus
}

type PeerManager struct {
  peers []Peer
  pieceRequestMessageBus *PieceRequestMessageBus,
  pieceRequestResponseMessageBus *PieceRequestResponseMessageBus,
}

type Peer struct {
  ...
  pieceRequestMessageBus *PieceRequestMessageBus,
  pieceResponseMessageBus *PieceRequestResponseMessageBus,
}

tm = TorrentManager {
  torrentFilePath: "some_path_passed",
  peers: []Peer{},
  pieceRequestMessageBus: pieceRequestMessageBus,
  pieceRequestResponseMessageBus: pieceRequestResponseMessageBus,
}

peerManager := PeerManager{
  peers: []Peer{},
  pieceRequestMessageBus: pieceRequestMessageBus,
  pieceRequestResponseMessageBus: pieceRequestResponseMessageBus,
}

peer := Peer {
  ....
  pieceRequestMessageBus: pieceRequestMessageBus,
  pieceRequestResponseBus: pieceRequestResponseBus,
}
```
torrentManager can send a command to peerManager and always listens for message from peerManager. It tells whether a piece has been downloaded.
Peer Manager can send a command to a peer to download a piece and receive a message from peer when it's done.


2. Peer Manager
3. Torrent Manager
4. Peer
