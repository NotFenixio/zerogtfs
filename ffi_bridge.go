package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"os"
	"strings"
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

	decoder := NewZeroGTFSEncoder(data)
	if err := decoder.Decode(source); err != nil {
		fmt.Printf("[Go FFI Error] Failed decoding binary format: %v\n", err)
		return 0
	}

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

// required to compile in shared c mode thing
func main() {}
