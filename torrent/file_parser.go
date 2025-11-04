package torrent

import (
	"os"

	"github.com/jackpal/bencode-go"
)

type torrentFile struct {
	path string
}

func (t torrentFile) Trackers() ([]string, error) {

}

func (t torrentFile) Parse() (interface{}, error) {
	file, err := os.Open(t.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := bencode.Decode(file)
	if err != nil {
		return nil, err
	}
	return data, nil
}
