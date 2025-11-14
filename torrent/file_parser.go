package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"strings"

	"github.com/jackpal/bencode-go"
)

type TorrentFile struct {
	Path string
}

type TorrentFileInfo struct {
	TorrentFile  *TorrentFile
	Info         map[string]any
	InfoHash     string
	HTTPTrackers []tracker
	UDPTrackers  []tracker
	Mode         fileType
	PieceLength  int64
	TotalPieces  int64
}

func (t TorrentFile) SetTorrentFileInfo() (TorrentFileInfo, error) {
	tfi := TorrentFileInfo{}

	info, err := t.Info()
	if err != nil {
		return tfi, err
	}

	parsedFile, err := t.Parse()
	if err != nil {
		return tfi, err
	}

	trackers, err := t.Trackers(parsedFile)
	if err != nil {
		return tfi, err
	}

	httpTrackers, err := t.HTTPTrackers(trackers)
	if err != nil {
		return tfi, err
	}

	udpTrackers, err := t.UDPTrackers(trackers)
	if err != nil {
		return tfi, err
	}

	infoHash, err := t.InfoHash(info)
	if err != nil {
		return tfi, err
	}

	fileMode := t.FileMode(info)
	pieceLength, ok := info["piece length"].(int64)
	if !ok {
		return tfi, fmt.Errorf("Piece length has to be a non-neg integer")
	}

	fileLength, ok := info["length"].(int64)
	if !ok {
		return tfi, fmt.Errorf("File length has to be asdfasa non-neg integer")
	}

	numberOfPieces := (fileLength + pieceLength - 1) / pieceLength // clever math trick to get ceil value

	tfi.TorrentFile = &t
	tfi.Info = info
	tfi.InfoHash = infoHash
	tfi.HTTPTrackers = httpTrackers
	tfi.UDPTrackers = udpTrackers
	tfi.Mode = fileMode
	tfi.PieceLength = pieceLength
	tfi.TotalPieces = numberOfPieces

	return tfi, nil
}

func (t TorrentFile) Trackers(dataMap map[string]any) ([]string, error) {
	trackers := []string{}
	trackersData, ok := dataMap["announce-list"].([]any)
	if !ok {
		return nil, fmt.Errorf("announce-list not found or not an array")
	}

	for _, val := range trackersData {
		v, ok := val.([]any)
		if !ok {
			return nil, fmt.Errorf("Not a nested array")
		}
		for _, g := range v {
			k, _ := g.(string)
			trackers = append(trackers, k)
		}
	}

	return trackers, nil
}

func (t TorrentFile) InfoHash(info map[string]any) (string, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, info)
	if err != nil {
		return "", err
	}

	sha1Array := sha1.Sum(buf.Bytes())
	return fmt.Sprintf("%x", sha1Array), nil
}

type fileType string

const single fileType = "single"
const multi fileType = "multi"

func (t TorrentFile) FileMode(info map[string]any) fileType {
	_, ok := info["files"]
	if ok {
		return multi
	} else {
		return single
	}
}

func (t TorrentFile) HTTPTrackers(trackers []string) ([]tracker, error) {
	var httpTrackers []tracker
	for _, str := range trackers {
		ok := strings.HasPrefix(str, "http")
		if ok {
			tr := tracker{
				kind: "http",
				url:  str,
			}
			httpTrackers = append(httpTrackers, tr)
		}
	}
	return httpTrackers, nil
}

func (t TorrentFile) UDPTrackers(trackers []string) ([]tracker, error) {
	var updTrackers []tracker
	for _, str := range trackers {
		ok := strings.HasPrefix(str, "udp")
		if ok {
			tr := tracker{
				kind: "udp",
				url:  str,
			}
			updTrackers = append(updTrackers, tr)
		}
	}
	return updTrackers, nil
}

func (t TorrentFile) Parse() (map[string]any, error) {
	file, err := os.Open(t.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := bencode.Decode(file)
	if err != nil {
		return nil, err
	}
	dataMap, ok := data.(map[string]any)
	if !ok {
		return nil, err
	}
	return dataMap, nil
}

func (t TorrentFile) Info() (map[string]any, error) {
	parsedFile, err := t.Parse()
	if err != nil {
		return nil, err
	}
	info, ok := parsedFile["info"].(map[string]any)
	if !ok {
		return nil, err
	}
	return info, nil
}
