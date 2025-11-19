This project was built to get hang of golang. AI(LLMs and Coding Agents) were
used heavily for brainstorming system design and debugging protocol
issues.

## System Components

1. Torrent Manager(Heart of the system)
2. Tracker Manager
3. Piece Manager
4. Peer Manager
5. DiskManager

### Torrent Manager

It orchestrates everything. I like to think of it as control plane of the
system as in k8s. All other components talk directly to it. And then it
takes some action against the message.

### Tracker Manager

It's job is to talk to trackers and get peers list. It also sets the peers
list by taking ref of peer manager.

### Piece Manager

Sort of librarian for pieces. Keeps tracks of all blocks, pieces and their
statuses.

### Peer Manager

Responsible for managing peers. Tracks idle peers and sends it to a channel.
Does CRUD around peers as well.

### Disk Manager

Writes blocks to file

## Channels

### IdlePeerBus - Carries idle peers that are ready to download blocks.
  
Producer: Peer manager's FindIdlePeers go routine scans all peers every 500ms and pushes idle ones here.
Consumer: Torrent manager listens on this channel and assigns work to idle peers.

### BlockRequestBus - Carries block download requests.

Producer: Torrent manager creates a BlockRequest (which peer should download which block) and pushes it here.
Consumer: Peer manager's ReadBlockRequestBus go routine picks up requests and spawns a go routine to handle each one.

### BlockRequestResponseBus - Carries downloaded block data.

Producer: Each peer's Listen loop receives block data from the network and pushes it here after parsing.
Consumer: Torrent manager receives the block data and hands it to disk manager.

### BlockWrittenBus - Carries disk write results (success or failure).

Producer: Disk manager pushes an event here after attempting to write a block to disk.
Consumer: Torrent manager handles the event by updating block status and checking if the piece is complete.


## Workflow

1. Parse the torrent file.
2. Init all components
3. Start a go routine to fetch peers. This is done by tracker manager. It sets the peers in peer manager as well.
4. After getting the peers we start the handshake process with each one of them in a separate go routine. If handshake succeeds we start a go routine to listen for messages against that peer.
5. Call init pieces and init blocks to create a map that tracks the download statuses of each of them. This is held by piece manager.
6. Scaffold files to be downloaded.
7. Start a go routine to find idle peers. It continuously finds idle peers and pushes it to a channel. The torrent manager continuously listens to that channel.
8. When an idle peer is received, torrent manager checks which pieces the peer has (using their bitfield) and picks a pending block to request from them.
9. The block request is pushed to the block request bus. Peer manager listens to this bus and spawns a go routine to handle each request.
10. Peer manager marks the peer as active and calls the peer's DownloadBlock method which sends a request message over TCP.
11. The peer's listen loop receives the block data in a piece message (type 7) and pushes it to the block response bus.
12. Torrent manager receives the block response and hands it off to disk manager to write the block to the correct file offset.
13. After writing, disk manager pushes a block written event to the block written bus.
14. Torrent manager handles this event by updating the block's status to "downloaded" and checking if all blocks in that piece are done.
15. If a piece is complete, it gets moved to the downloaded state and progress is printed. The peer goes back to idle and the cycle continues until all pieces are downloaded.
