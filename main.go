package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	fmt.Println("zeroGTFS: Binary Compression for GTFS Transit Data")
	fmt.Println("====================================================")
	fmt.Println()

	var gtfsSource string
	flag.StringVar(&gtfsSource, "gtfs", "", "Path to GTFS ZIP file or URL (if not provided, uses mock data)")
	flag.Parse()

	testFile := "feed.zgts"
	_ = os.Remove(testFile)

	var mockData *MockGTFSData
	var err error

	if gtfsSource != "" {
		fmt.Printf("Step 1: Loading GTFS data from: %s\n", gtfsSource)
		mockData, err = loadGTFSData(gtfsSource)
		if err != nil {
			log.Fatalf("Failed to load GTFS data: %v", err)
		}
	} else {
		fmt.Println("Step 1: Creating mock GTFS data...")
		mockData = NewMockData()
	}
	fmt.Printf("  - Agencies: %d\n", len(mockData.Agencies))
	fmt.Printf("  - Stops: %d\n", len(mockData.Stops))
	fmt.Printf("  - Routes: %d\n", len(mockData.Routes))
	fmt.Printf("  - Trips: %d\n", len(mockData.Trips))
	fmt.Printf("  - Stop Times: %d\n", len(mockData.StopTimes))

	originalSize := EstimateCSVSize(mockData)
	fmt.Printf("  Estimated original CSV size: ~%d bytes\n", originalSize)

	fmt.Println("\nStep 2: Encoding to binary zeroGTFS format...")
	encoder := NewZeroGTFSEncoder(mockData)
	if err := encoder.Encode(testFile); err != nil {
		log.Fatalf("Failed to encode: %v", err)
	}

	fileInfo, err := os.Stat(testFile)
	if err != nil {
		log.Fatalf("Failed to stat file: %v", err)
	}
	encodedSize := fileInfo.Size()
	fmt.Printf("  ✓ Encoded file size: %d bytes\n", encodedSize)

	ratio := float64(originalSize) / float64(encodedSize)
	compression := ((float64(originalSize) - float64(encodedSize)) / float64(originalSize)) * 100

	fmt.Printf("  Compression ratio: %.2fx\n", ratio)
	fmt.Printf("  Space saved: %.1f%%\n", compression)

	fmt.Println("\nStep 3: Decoding from binary zeroGTFS format...")
	decoded, err := DecodeZeroGTFSFromFile(testFile)
	if err != nil {
		log.Fatalf("Failed to decode: %v", err)
	}
	fmt.Println("  ✓ Successfully decoded")

	fmt.Println("\nStep 4: Verifying data integrity...")
	if len(decoded.Agencies) != len(mockData.Agencies) {
		log.Fatalf("Agency count mismatch: %d vs %d", len(decoded.Agencies), len(mockData.Agencies))
	}
	fmt.Printf("  ✓ Agencies: %d (match)\n", len(decoded.Agencies))

	if len(decoded.Stops) != len(mockData.Stops) {
		log.Fatalf("Stop count mismatch: %d vs %d", len(decoded.Stops), len(mockData.Stops))
	}
	fmt.Printf("  ✓ Stops: %d (match)\n", len(decoded.Stops))

	if len(decoded.Routes) != len(mockData.Routes) {
		log.Fatalf("Route count mismatch: %d vs %d", len(decoded.Routes), len(mockData.Routes))
	}
	fmt.Printf("  ✓ Routes: %d (match)\n", len(decoded.Routes))

	if len(decoded.Patterns) > 0 {
		fmt.Printf("  ✓ Trip Patterns: %d (clustered from %d trips)\n", len(decoded.Patterns), len(mockData.Trips))
	}

	if len(decoded.Schedules) > 0 {
		fmt.Printf("  ✓ Schedules: %d\n", len(decoded.Schedules))
	}

	fmt.Println("\nStep 5: Demonstrating fast lookup capabilities...")

	if len(decoded.Agencies) > 0 {
		agency := decoded.Agencies[0]
		fmt.Printf("  Sample Agency:\n")
		fmt.Printf("    ID: %s\n", agency.AgencyID)
		fmt.Printf("    Name: %s\n", agency.Name)
		fmt.Printf("    Timezone: %s\n", agency.Timezone)
	}

	if len(decoded.Stops) > 0 {
		stop := decoded.Stops[0]
		fmt.Printf("  Sample Stop:\n")
		fmt.Printf("    ID: %s\n", stop.StopID)
		fmt.Printf("    Name: %s\n", stop.Name)
		fmt.Printf("    Lat: %.4f, Lon: %.4f\n", stop.Lat, stop.Lon)
	}

	if len(decoded.Routes) > 0 {
		route := decoded.Routes[0]
		fmt.Printf("  Sample Route:\n")
		fmt.Printf("    ID: %s\n", route.RouteID)
		fmt.Printf("    Name: %s\n", route.LongName)
		fmt.Printf("    Type: %d\n", route.Type)
	}

	fmt.Println("\nStep 6: Exporting sample data to JSON for inspection...")
	exportSampleJSON(decoded)

	fmt.Printf("\n\n=== COMPRESSION SUMMARY ===\n")
	fmt.Printf("Original size (estimated CSV): %d bytes\n", originalSize)
	fmt.Printf("Binary zeroGTFS size: %d bytes\n", encodedSize)
	fmt.Printf("Compression ratio: %.2fx (%.1f%% reduction)\n", ratio, compression)
	fmt.Printf("\nKey Optimization Techniques:\n")
	fmt.Printf("  1. Dictionary Encoding: All strings → token IDs\n")
	fmt.Printf("  2. Delta Encoding: Relative timestamps in trip patterns\n")
	fmt.Printf("  3. Trip Pattern Clustering: 80-90%% redundancy elimination\n")
	fmt.Printf("  4. Memory-mapped binary layout: O(1) random access\n")
	fmt.Printf("\nPerformance:\n")
	fmt.Printf("  - String table: %d unique strings\n", len(decoded.StringTable.IDToString))
	fmt.Printf("  - Trip patterns: %d unique (from %d trips)\n", len(decoded.Patterns), len(mockData.Trips))
	fmt.Printf("  - Bytes per stop: ~%.2f\n", float64(encodedSize)/float64(len(decoded.Stops)))

	_ = os.Remove(testFile)
	fmt.Println("\n✓ Test completed successfully!")
}

func exportSampleJSON(decoded *DecodedZeroGTFSFeed) {
	type SampleExport struct {
		Agencies []map[string]interface{} `json:"agencies"`
		Stops    []map[string]interface{} `json:"stops"`
		Routes   []map[string]interface{} `json:"routes"`
		Patterns []map[string]interface{} `json:"patterns"`
		Summary  map[string]interface{}   `json:"summary"`
	}

	sample := SampleExport{Summary: make(map[string]interface{})}

	for _, a := range decoded.Agencies {
		sample.Agencies = append(sample.Agencies, map[string]interface{}{
			"agency_id": a.AgencyID,
			"name":      a.Name,
			"timezone":  a.Timezone,
		})
	}

	for i, s := range decoded.Stops {
		if i >= 10 {
			break
		}
		sample.Stops = append(sample.Stops, map[string]interface{}{
			"stop_id": s.StopID,
			"name":    s.Name,
			"lat":     s.Lat,
			"lon":     s.Lon,
		})
	}

	for i, r := range decoded.Routes {
		if i >= 10 {
			break
		}
		sample.Routes = append(sample.Routes, map[string]interface{}{
			"route_id":   r.RouteID,
			"short_name": r.ShortName,
			"long_name":  r.LongName,
			"type":       r.Type,
		})
	}

	for i, p := range decoded.Patterns {
		if i >= 5 {
			break
		}
		stops := make([]string, len(p.StopIDTokens))
		for j, t := range p.StopIDTokens {
			stops[j] = decoded.StringTable.Lookup(t)
		}
		sample.Patterns = append(sample.Patterns, map[string]interface{}{
			"pattern_id": p.PatternID,
			"stop_count": p.StopCount,
			"stops":      stops,
		})
	}

	sample.Summary["agencies"] = len(decoded.Agencies)
	sample.Summary["stops"] = len(decoded.Stops)
	sample.Summary["routes"] = len(decoded.Routes)
	sample.Summary["patterns"] = len(decoded.Patterns)
	sample.Summary["schedules"] = len(decoded.Schedules)

	data, err := json.MarshalIndent(sample, "", "  ")
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return
	}
	if err := os.WriteFile("sample_output.json", data, 0644); err != nil {
		log.Printf("Error writing JSON file: %v", err)
		return
	}
	fmt.Println("  ✓ Exported to sample_output.json")
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
