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
    1. Start a go routing whose job is to find a idle active peer. Get a piece from piece manager
       and ask the torrent manager to download it.(Ask for a Piece Manager)
    2. Start a event loop to handle incoming messages from Peer Manager, Ask for a Piece Manager, Trigger Piece Request to
       Peer Manager.
    3. When you receive piece from torrent manager, ask disk manager to save it.
