package zerogtfs

import (
	lib "zerogtfs/lib"
)

// Mobile wrappers for gomobile binding - re-exports of lib functions with pure Go interface
// These functions have no cgo dependencies and can be processed by gomobile bind

// NewMockData creates mock GTFS transit data
func NewMockData() *lib.MockGTFSData {
	return lib.NewMockData()
}

// NewZeroGTFSEncoder creates a new encoder for the given GTFS data
func NewZeroGTFSEncoder(data *lib.MockGTFSData) *lib.ZeroGTFSEncoder {
	return lib.NewZeroGTFSEncoder(data)
}

// Encode encodes the GTFS data to a zeroGTFS binary file
func EncodeToFile(data *lib.MockGTFSData, filename string) error {
	encoder := lib.NewZeroGTFSEncoder(data)
	return encoder.Encode(filename)
}

// DecodeZeroGTFSFromFile decodes a zeroGTFS binary file back to structured data
func DecodeZeroGTFSFromFile(filename string) (*lib.DecodedZeroGTFSFeed, error) {
	return lib.DecodeZeroGTFSFromFile(filename)
}

// ReadGTFSFromZip reads a GTFS ZIP file and returns the structured data
func ReadGTFSFromZip(zipPath string) (*lib.MockGTFSData, error) {
	return lib.ReadGTFSFromZip(zipPath)
}

// ExportGTFSZip exports decoded zeroGTFS data back to GTFS ZIP format
func ExportGTFSZip(feed *lib.DecodedZeroGTFSFeed, outputFile string) error {
	return lib.ExportGTFSZip(feed, outputFile)
}

// PerformRoundtrip performs a complete roundtrip: load GTFS -> encode -> decode -> export
func PerformRoundtrip(source, encodedOutput, decodedOutput string) error {
	return lib.PerformRoundtrip(source, encodedOutput, decodedOutput)
}

// EstimateCSVSize estimates the uncompressed CSV size of the GTFS data
func EstimateCSVSize(mockData *lib.MockGTFSData) int {
	return lib.EstimateCSVSize(mockData)
}

// TimeToSeconds converts HH:MM:SS format to seconds past midnight
func TimeToSeconds(hhmmss string) uint32 {
	return lib.TimeToSeconds(hhmmss)
}

// SecondsToTime converts seconds past midnight to HH:MM:SS format
func SecondsToTime(seconds uint32) string {
	return lib.SecondsToTime(seconds)
}

// GetStopsByTrip retrieves stops for a given trip from decoded data
func GetStopsByTrip(feed *lib.DecodedZeroGTFSFeed, tripIndex uint32) ([]lib.TripStopInfo, error) {
	return feed.GetStopsByTrip(tripIndex)
}
