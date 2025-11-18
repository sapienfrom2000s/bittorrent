package torrent

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const basePath = "/Users/thirtyone/asdf/"

type filesMap struct {
	filesData []fileData
}

type fileData struct {
	filePath    string
	fileSize    int64
	offsetStart int64
	offsetEnd   int64
}

// Maybe no need to pass the whole ref, just pass info(pass by value)
type DiskManager struct {
	TorrentFileInfo *TorrentFileInfo
	mu              sync.Mutex
	filesMap        *filesMap
	BlockWrittenBus *BlockWrittenBus
}

func (diskManager *DiskManager) saveBlock(blockResponse *BlockResponse) {
	diskManager.mu.Lock()
	defer diskManager.mu.Unlock()

	offset := diskManager.TorrentFileInfo.PieceLength*int64(blockResponse.pieceIndex) + int64(blockResponse.blockIndex)*blockLength
	var pathToWrite string
	var fileOffset int64

	for _, fileData := range diskManager.filesMap.filesData {
		if (offset >= fileData.offsetStart) && (offset < fileData.offsetEnd) {
			pathToWrite = fileData.filePath
			fileOffset = offset - fileData.offsetStart
		}
	}

	file, err := os.OpenFile(pathToWrite, os.O_RDWR, 0777)
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", pathToWrite, err)
		return
	}
	defer file.Close()

	_, err = file.Seek(fileOffset, 0)
	if err != nil {
		fmt.Printf("Error seeking in file %s at offset %d: %v\n", pathToWrite, fileOffset, err)
		return
	}

	_, err = file.Write(blockResponse.blockData)
	if err != nil {
		fmt.Printf("Error writing to file %s: %v\n", pathToWrite, err)
		return
	}

	// Send success event
	diskManager.BlockWrittenBus.BlockWritten <- &BlockWritten{
		pieceIndex: blockResponse.pieceIndex,
		blockIndex: blockResponse.blockIndex,
		success:    true,
		err:        nil,
	}
}

func (diskManager *DiskManager) ScaffoldFiles() {
	fileType := diskManager.TorrentFileInfo.Mode
	filesMap := &filesMap{}
	diskManager.filesMap = filesMap

	switch fileType {
	case "single":
		fileName := diskManager.TorrentFileInfo.Info["name"].(string)
		fileSize, ok := diskManager.TorrentFileInfo.Info["length"].(int)
		fullPath := fmt.Sprintf("%s%s", basePath, fileName)
		if !ok {
			fmt.Println("File size has to be present")
		}
		createFile(fullPath, fileSize)
		fileData := fileData{
			filePath:    fileName,
			fileSize:    int64(fileSize),
			offsetStart: 0,
			offsetEnd:   int64(fileSize),
		}
		filesMap.filesData = append(filesMap.filesData, fileData)
	case "multi":
		files := diskManager.TorrentFileInfo.Info["files"]

		lastOffsetEnd := 0
		for _, file := range files.([]any) {
			f := file.(map[string]any)
			fileSize := f["length"].(int)
			path := f["path"].(string)
			fullPath := fmt.Sprintf("%s%s", basePath, path)
			os.MkdirAll(filepath.Dir(fullPath), 0777)
			createFile(fullPath, fileSize)
			fileData := fileData{
				filePath:    path,
				fileSize:    int64(fileSize),
				offsetStart: int64(lastOffsetEnd),
				offsetEnd:   int64(lastOffsetEnd + fileSize),
			}
			lastOffsetEnd = lastOffsetEnd + fileSize
			filesMap.filesData = append(filesMap.filesData, fileData)
		}
	default:
		panic("FileType has to be single or multi")
	}
}

func createFile(filePath string, fileSize int) error {
	buff := make([]byte, fileSize)
	err := os.WriteFile(filePath, buff, 0777)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
