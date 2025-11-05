package torrent

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jackpal/bencode-go"
)

type torrentFile struct {
	path string
}

func (t torrentFile) Trackers(dataMap map[string]any) ([]string, error) {
	trackers, ok := dataMap["announce-list"].([]string)
	if !ok {
		return nil, fmt.Errorf("announce-list not in array of strings")
	}

	return trackers, nil
}

func (t torrentFile) InfoHash(info map[string]any) (string, error) {
	bytes, err := json.Marshal(info)
	if err != nil {
		return "", fmt.Errorf("Unable to read marshal info")
	}

	sha1Array := sha1.Sum(bytes)
	return fmt.Sprintf("%x", sha1Array), nil
}

type fileType string

const single fileType = "single"
const multi fileType = "multi"

func (t torrentFile) FileMode(info map[string]any) fileType {
	_, ok := info["files"]
	if ok {
		return multi
	} else {
		return single
	}
}

func (t torrentFile) HTTPTrackers(trackers []string) ([]string, error) {
	var httpTrackers []string
	for _, tracker := range trackers {
		ok := strings.HasPrefix(tracker, "http")
		if ok {
			httpTrackers = append(httpTrackers, tracker)
		}
	}
	return httpTrackers, nil
}

func (t torrentFile) UDPTrackers(trackers []string) ([]string, error) {
	var updTrackers []string
	for _, tracker := range trackers {
		ok := strings.HasPrefix(tracker, "udp")
		if ok {
			updTrackers = append(updTrackers, tracker)
		}
	}
	return updTrackers, nil
}

func (t torrentFile) Parse() (map[string]any, error) {
	file, err := os.Open(t.path)
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
		return nil, fmt.Errorf("File has to be in dictionary format")
	}
	return dataMap, nil
}
