This project was built to get hang of golang. AI(LLMs and Coding Agents) were
used heavily for brainstorming system design and debugging protocol
issues.

## How to Run

1. Place your `.torrent` file in the `torrent/` directory (e.g., `torrent/test.torrent`).
Test torrent will also work fine. To view the contents of the file you can use `https://chocobo1.github.io/bencode_online/`
2. Update the torrent file path in `main.go` if needed:
   ```go
   tf := torrent.TorrentFile{
       Path: "torrent/test.torrent",
   }
   ```
3. (Optional) Change the download directory by modifying `basePath` in `torrent/disk_manager.go`:
   ```go
   const basePath = "./asdf/"  // Change this to your preferred location
   ```
4. Build and run:
   ```bash
   go run main.go
   ```

Downloaded files will be saved in the directory specified by `basePath` (default: `./asdf/`).

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


## Sample Output

```
➜  bittorrent git:(main) ✗ go run main.go
=== Torrent File Information ===
InfoHash: 3c315a7c4a929aa088a26ba075c1ae7703fca4ca
Mode: single
File Length: 195137002 bytes (186.10 MB)
Piece Length: 16777216 bytes
Total Pieces: 12

=== Trackers ===
1. [udp] udp://tracker.opentrackr.org:1337/announce
2. [udp] udp://open.demonii.com:1337/announce
3. [udp] udp://open.stealth.si:80/announce
4. [udp] udp://tracker.tiny-vps.com:6969/announce
5. [udp] udp://tracker.torrent.eu.org:451/announce
6. [udp] udp://explodie.org:6969/announce
7. [udp] udp://exodus.desync.com:6969/announce
8. [udp] udp://p4p.arenabg.com:1337/announce
9. [udp] udp://tracker.dler.org:6969/announce
10. [udp] udp://movies.zsw.ca:6969/announce
11. [udp] udp://tracker.openbittorrent.com:6969/announce
12. [udp] udp://uploads.gamecoast.net:6969/announce
13. [udp] udp://tracker.cyberia.is:6969/announce
14. [udp] udp://ipv4.tracker.harry.lu:80/announce
15. [udp] udp://ipv6.tracker.harry.lu:80/announce
16. [udp] udp://tracker1.bt.moack.co.kr:80/announce
17. [udp] udp://opentracker.i2p.rocks:6969/announce
18. [udp] udp://eddie4.nl:6969/announce
19. [udp] udp://bt1.archive.org:6969/announce
20. [udp] udp://tracker.swateam.org.uk:2710/announce
21. [http] http://tracker.openbittorrent.com:80/announce
22. [http] http://tracker.opentrackr.org:1337/announce
23. [http] https://tracker1.520.jp:443/announce
24. [http] https://tracker.tamersunion.org:443/announce
25. [http] https://tracker.imgoingto.icu:443/announce
26. [http] http://nyaa.tracker.wf:7777/announce
27. [udp] udp://tracker2.dler.org:80/announce
28. [udp] udp://tracker.theoks.net:6969/announce
29. [udp] udp://tracker.dump.cl:6969/announce
30. [udp] udp://tracker.bittor.pw:1337/announce
31. [udp] udp://tracker.4.babico.name.tr:3131/announce
32. [udp] udp://sanincode.com:6969/announce
33. [udp] udp://retracker01-msk-virt.corbina.net:80/announce
34. [udp] udp://private.anonseed.com:6969/announce
35. [udp] udp://open.free-tracker.ga:6969/announce
36. [udp] udp://isk.richardsw.club:6969/announce
37. [udp] udp://htz3.noho.st:6969/announce
38. [udp] udp://epider.me:6969/announce
39. [udp] udp://bt.ktrackers.com:6666/announce
40. [udp] udp://acxx.de:6969/announce
41. [udp] udp://aarsen.me:6969/announce
42. [udp] udp://6ahddutb1ucc3cp.ru:6969/announce
43. [udp] udp://yahor.of.by:6969/announce
44. [udp] udp://v2.iperson.xyz:6969/announce
45. [udp] udp://tracker1.myporn.club:9337/announce
46. [udp] udp://tracker.therarbg.com:6969/announce
47. [udp] udp://tracker.qu.ax:6969/announce
48. [udp] udp://tracker.publictracker.xyz:6969/announce
49. [udp] udp://tracker.netmap.top:6969/announce
50. [udp] udp://tracker.farted.net:6969/announce
51. [udp] udp://tracker.cubonegro.lol:6969/announce
52. [udp] udp://tracker.ccp.ovh:6969/announce
53. [udp] udp://tracker.0x7c0.com:6969/announce
54. [udp] udp://thouvenin.cloud:6969/announce
55. [udp] udp://thinking.duckdns.org:6969/announce
56. [udp] udp://tamas3.ynh.fr:6969/announce
57. [udp] udp://ryjer.com:6969/announce
58. [udp] udp://run.publictracker.xyz:6969/announce
59. [udp] udp://run-2.publictracker.xyz:6969/announce
60. [udp] udp://public.tracker.vraphim.com:6969/announce
61. [udp] udp://public.publictracker.xyz:6969/announce
62. [udp] udp://public-tracker.cf:6969/announce
63. [udp] udp://opentracker.io:6969/announce
64. [udp] udp://open.u-p.pw:6969/announce
65. [udp] udp://open.dstud.io:6969/announce
66. [udp] udp://oh.fuuuuuck.com:6969/announce
67. [udp] udp://new-line.net:6969/announce
68. [udp] udp://moonburrow.club:6969/announce
69. [udp] udp://mail.segso.net:6969/announce
70. [udp] udp://free.publictracker.xyz:6969/announce
71. [udp] udp://carr.codes:6969/announce
72. [udp] udp://bt2.archive.org:6969/announce
73. [udp] udp://6.pocketnet.app:6969/announce
74. [udp] udp://1c.premierzal.ru:6969/announce
75. [udp] udp://tracker.t-rb.org:6969/announce
76. [udp] udp://tracker.srv00.com:6969/announce
77. [udp] udp://tracker.artixlinux.org:6969/announce
78. [udp] udp://tracker-udp.gbitt.info:80/announce
79. [udp] udp://torrents.artixlinux.org:6969/announce
80. [udp] udp://psyco.fr:6969/announce
81. [udp] udp://mail.artixlinux.org:6969/announce
82. [udp] udp://lloria.fr:6969/announce
83. [udp] udp://fh2.cmp-gaming.com:6969/announce
84. [udp] udp://concen.org:6969/announce
85. [udp] udp://boysbitte.be:6969/announce
86. [udp] udp://aegir.sexy:6969/announce

=== Info Dictionary Keys ===
- length
- name
- piece length
- pieces

 Torrent file parsed successfully!


 Starting download...
 UDP tracker: tracker.opentrackr.org:1337
 Scaffolding files (mode: single)...
 Created file: ./asdf/Smiling.Friends.S03E01.Silly.Samuel.1080p.MAX.WEB-DL.DDP5.1.H.264-ViETNAM.mkv (size: 195137002 bytes)
 Starting block request bus reader...
 Starting idle peer finder...
 → 200 peers
  Added 200 new peers (total: 200)

 Got 200 peers, skipping remaining 85 trackers


 Total unique peers: 200

 Handshake successful with peer 138.199.33.238
 Sent 'interested' message to peer 138.199.33.238
 Connected to peer: 138.199.33.238
 Received bitfield from peer 138.199.33.238 (2 bytes)
 DEBUG: Bitfield stored, length=2, unchoked=false
 Handshake successful with peer 124.169.242.186
 Sent 'interested' message to peer 124.169.242.186
 Connected to peer: 124.169.242.186
 Received bitfield from peer 124.169.242.186 (2 bytes)
 DEBUG: Bitfield stored, length=2, unchoked=false
 Handshake successful with peer 195.214.255.228
 Sent 'interested' message to peer 195.214.255.228
 Connected to peer: 195.214.255.228
 Received bitfield from peer 195.214.255.228 (2 bytes)
 DEBUG: Bitfield stored, length=2, unchoked=false
 Peer 195.214.255.228 unchoked us - ready to download!
 Peer 195.214.255.228 is now ready (has bitfield + unchoked)
 Handshake successful with peer 82.11.7.127
 Sent 'interested' message to peer 82.11.7.127
 Connected to peer: 82.11.7.127
 Received bitfield from peer 82.11.7.127 (2 bytes)
 DEBUG: Bitfield stored, length=2, unchoked=false
 Handshake successful with peer 212.32.244.67
 Sent 'interested' message to peer 212.32.244.67
 Connected to peer: 212.32.244.67
 Handshake successful with peer 71.17.17.244
 Sent 'interested' message to peer 71.17.17.244
 Connected to peer: 71.17.17.244
 Received bitfield from peer 71.17.17.244 (2 bytes)
 DEBUG: Bitfield stored, length=2, unchoked=false
 Handshake successful with peer 158.173.21.194
 Unknown message type 4 from peer 71.17.17.244
 Unknown message type 4 from peer 71.17.17.244
 Sent 'interested' message to peer 158.173.21.194
 Unknown message type 4 from peer 71.17.17.244
 Connected to peer: 158.173.21.194
 Received bitfield from peer 158.173.21.194 (2 bytes)
 DEBUG: Bitfield stored, length=2, unchoked=false
 Peer 158.173.21.194 unchoked us - ready to download!
 Peer 158.173.21.194 is now ready (has bitfield + unchoked)
 Handshake successful with peer 184.75.208.246
 Sent 'interested' message to peer 184.75.208.246
 Connected to peer: 184.75.208.246
 Received bitfield from peer 184.75.208.246 (2 bytes)
 DEBUG: Bitfield stored, length=2, unchoked=false
 Handshake successful with peer 95.146.232.37
 Sent 'interested' message to peer 95.146.232.37
 Connected to peer: 95.146.232.37
 Received bitfield from peer 95.146.232.37 (2 bytes)
 DEBUG: Bitfield stored, length=2, unchoked=false
 Handshake successful with peer 185.98.168.5
 Sent 'interested' message to peer 185.98.168.5
 Connected to peer: 185.98.168.5
 Received bitfield from peer 185.98.168.5 (2 bytes)
 DEBUG: Bitfield stored, length=2, unchoked=false
 Handshake successful with peer 66.56.81.96
 Sent 'interested' message to peer 66.56.81.96
 Connected to peer: 66.56.81.96
 Received bitfield from peer 66.56.81.96 (2 bytes)
 DEBUG: Bitfield stored, length=2, unchoked=false
 Handshake successful with peer 66.56.81.110
 Sent 'interested' message to peer 66.56.81.110
 Connected to peer: 66.56.81.110
 Checking 12 pending pieces against peer 195.214.255.228 bitfield
 Requesting block (piece=11, block=0) from peer 195.214.255.228
 Checking 12 pending pieces against peer 158.173.21.194 bitfield
 Requesting block (piece=9, block=0) from peer 158.173.21.194
 Found 2 idle peer(s), sending to bus
 Processing block request (piece=11, block=0) for peer 195.214.255.228
 Processing block request (piece=9, block=0) for peer 158.173.21.194
 Sending request: piece=9, begin=0, length=16384
 Sending request: piece=11, begin=0, length=16384
 Handshake successful with peer 212.32.48.147
 Sent 'interested' message to peer 212.32.48.147
 Connected to peer: 212.32.48.147
 Received bitfield from peer 212.32.48.147 (2 bytes)
 DEBUG: Bitfield stored, length=2, unchoked=false
 Peer 212.32.48.147 unchoked us - ready to download!
 Peer 212.32.48.147 is now ready (has bitfield + unchoked)
 Handshake successful with peer 213.152.161.52
 Sent 'interested' message to peer 213.152.161.52
 Connected to peer: 213.152.161.52
 Handshake successful with peer 158.173.21.223
 Received bitfield from peer 213.152.161.52 (2 bytes)
 DEBUG: Bitfield stored, length=2, unchoked=false
 Sent 'interested' message to peer 158.173.21.223
 Connected to peer: 158.173.21.223
 Received bitfield from peer 158.173.21.223 (2 bytes)
 DEBUG: Bitfield stored, length=2, unchoked=false
 Handshake successful with peer 68.235.52.68
 Sent 'interested' message to peer 68.235.52.68
 Connected to peer: 68.235.52.68
 Received bitfield from peer 68.235.52.68 (2 bytes)
 DEBUG: Bitfield stored, length=2, unchoked=false
 Peer 68.235.52.68 unchoked us - ready to download!
 Peer 68.235.52.68 is now ready (has bitfield + unchoked)
```
