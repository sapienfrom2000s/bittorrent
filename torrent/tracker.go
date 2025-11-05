package torrent

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/jackpal/bencode-go"
)

type tracker struct {
	url  string
	kind string
}

func (t tracker) Peers(infoHash string) ([]any, error) {
	switch t.kind {
	case "http":
		return t.httpTrackerPeers(infoHash)
	case "udp":
		return nil, fmt.Errorf("udp not handled")
	}
	return nil, fmt.Errorf("only works for udp and http")
}

func (t tracker) httpTrackerPeers(infoHash string) ([]any, error) {
	params := url.Values{}
	params.Add("info_hash", infoHash)
	params.Add("peer_id", "abcde12345abcde12345")
	params.Add("port", "6883")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", "10000")
	params.Add("compact", "0")

	fullURL := t.url + "?" + params.Encode()
	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("Unable to get response from http tracker server")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http Server didn't throw 200")
	}

	dataMap, err := bencode.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Unable to decode response body")
	}

	dict, ok := dataMap.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("tracker response not in dictionary format")
	}

	peersMap, ok := dict["peers"].([]any)
	if !ok {
		return nil, fmt.Errorf("Peers map not in dictionary format")
	}

	return peersMap, nil
}
