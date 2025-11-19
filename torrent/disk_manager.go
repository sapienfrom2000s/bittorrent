package torrent

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const basePath = "./asdf/"

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

	fmt.Printf(" Writing block to disk (piece=%d, block=%d, size=%d bytes)\n",
		blockResponse.pieceIndex, blockResponse.blockIndex, len(blockResponse.blockData))

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
		fmt.Printf("❌ Error opening file %s: %v\n", pathToWrite, err)
		diskManager.BlockWrittenBus.BlockWritten <- &BlockWritten{
			pieceIndex: blockResponse.pieceIndex,
			blockIndex: blockResponse.blockIndex,
			success:    false,
			err:        err,
		}
		return
	}
	defer file.Close()

	_, err = file.Seek(fileOffset, 0)
	if err != nil {
		fmt.Printf("❌ Error seeking in file %s at offset %d: %v\n", pathToWrite, fileOffset, err)
		diskManager.BlockWrittenBus.BlockWritten <- &BlockWritten{
			pieceIndex: blockResponse.pieceIndex,
			blockIndex: blockResponse.blockIndex,
			success:    false,
			err:        err,
		}
		return
	}

	bytesWritten, err := file.Write(blockResponse.blockData)
	if err != nil {
		fmt.Printf("❌ Error writing to file %s: %v\n", pathToWrite, err)
		diskManager.BlockWrittenBus.BlockWritten <- &BlockWritten{
			pieceIndex: blockResponse.pieceIndex,
			blockIndex: blockResponse.blockIndex,
			success:    false,
			err:        err,
		}
		return
	}

	fmt.Printf(" Successfully wrote %d bytes to %s at offset %d\n", bytesWritten, pathToWrite, fileOffset)

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

	// Create base directory if it doesn't exist
	err := os.MkdirAll(basePath, 0777)
	if err != nil {
		fmt.Printf("❌ Failed to create base directory %s: %v\n", basePath, err)
		return
	}

	fmt.Printf("\n Scaffolding files (mode: %s)...\n", fileType)

	switch fileType {
	case "single":
		fileName := diskManager.TorrentFileInfo.Info["name"].(string)
		fileSize, ok := diskManager.TorrentFileInfo.Info["length"].(int64)
		fullPath := fmt.Sprintf("%s%s", basePath, fileName)
		if !ok {
			fmt.Println("❌ File size has to be present")
			return
		}
		err := createFile(fullPath, int(fileSize))
		if err != nil {
			return
		}
		fileData := fileData{
			filePath:    fullPath,
			fileSize:    fileSize,
			offsetStart: 0,
			offsetEnd:   fileSize,
		}
		filesMap.filesData = append(filesMap.filesData, fileData)
	case "multi":
		files := diskManager.TorrentFileInfo.Info["files"]

		lastOffsetEnd := int64(0)
		for _, file := range files.([]any) {
			f := file.(map[string]any)
			fileSize := f["length"].(int64)

			// Get path - it's an array of path components
			pathComponents, ok := f["path"].([]any)
			if !ok {
				continue
			}

			// Build path from components
			pathParts := make([]string, len(pathComponents))
			for i, component := range pathComponents {
				pathParts[i] = component.(string)
			}
			path := filepath.Join(pathParts...)

			fullPath := filepath.Join(basePath, path)
			os.MkdirAll(filepath.Dir(fullPath), 0777)
			err := createFile(fullPath, int(fileSize))
			if err != nil {
				continue
			}

			fileData := fileData{
				filePath:    fullPath,
				fileSize:    fileSize,
				offsetStart: lastOffsetEnd,
				offsetEnd:   lastOffsetEnd + fileSize,
			}
			lastOffsetEnd = lastOffsetEnd + fileSize
			filesMap.filesData = append(filesMap.filesData, fileData)
		}
	default:
		panic("FileType has to be single or multi")
	}
}

func createFile(filePath string, fileSize int) error {
	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("❌ Failed to create file %s: %v\n", filePath, err)
		return err
	}
	defer file.Close()

	// create a file of size n
	err = file.Truncate(int64(fileSize))
	if err != nil {
		fmt.Printf("❌ Failed to set file size for %s: %v\n", filePath, err)
		return err
	}

	fmt.Printf(" Created file: %s (size: %d bytes)\n", filePath, fileSize)
	return nil
}
