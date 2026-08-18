package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	decodedFeed      *DecodedZeroGTFSFeed
	decodedFeedMutex sync.Mutex
)

//export GenerateZeroGTFS
func GenerateZeroGTFS(cSource *C.char, cOutput *C.char) C.int {
	source := C.GoString(cSource)
	outputFile := C.GoString(cOutput)

	var data *MockGTFSData
	var err error

	if source != "" && source != "mock" {
		data, err = loadGTFSData(source)
		if err != nil {
			fmt.Printf("[Go FFI Error] Failed loading GTFS: %v\n", err)
			return 0
		}
	} else {
		data = NewMockData()
	}

	encoder := NewZeroGTFSEncoder(data)
	if err := encoder.Encode(outputFile); err != nil {
		fmt.Printf("[Go FFI Error] Failed encoding binary format: %v\n", err)
		return 0
	}

	return 1 // success!
}

//export DecodeZeroGTFS
func DecodeZeroGTFS(cSource *C.char) C.int {
	source := C.GoString(cSource)

	decoded, err := DecodeZeroGTFSFromFile(source)
	if err != nil {
		fmt.Printf("[Go FFI Error] Failed decoding binary format: %v\n", err)
		return 0
	}

	decodedFeedMutex.Lock()
	decodedFeed = decoded
	decodedFeedMutex.Unlock()
	return 1 // success!
}

func loadGTFSData(source string) (*MockGTFSData, error) {

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		zipPath := "gtfs_feed.zip"
		if err := DownloadFile(source, zipPath); err != nil {
			return nil, err
		}
		defer os.Remove(zipPath)
		return ReadGTFSFromZip(zipPath)
	}

	return ReadGTFSFromZip(source)
}

//export ExportDecodedToGTFS
func ExportDecodedToGTFS(cOutput *C.char) C.int {
	outputFile := C.GoString(cOutput)

	decodedFeedMutex.Lock()
	feed := decodedFeed
	decodedFeedMutex.Unlock()

	if feed == nil {
		fmt.Printf("[Go FFI Error] No decoded data available\n")
		return 0
	}

	if err := exportGTFSZip(feed, outputFile); err != nil {
		fmt.Printf("[Go FFI Error] Failed exporting to GTFS: %v\n", err)
		// Cleanup partial file on error
		if err := os.Remove(outputFile); err != nil {
			fmt.Printf("[Go FFI Warning] Failed to cleanup partial file: %v\n", err)
		}
		return 0
	}

	return 1 // success!
}

//export FreeDecodedFeed
func FreeDecodedFeed() {
	decodedFeedMutex.Lock()
	decodedFeed = nil
	decodedFeedMutex.Unlock()
}

// required to compile in shared c mode thing
func main() {}
