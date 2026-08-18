package main

import (
	"fmt"
	"os"
)

// PerformRoundtrip performs a complete roundtrip: load GTFS -> encode -> decode -> export
func PerformRoundtrip(source, encodedOutput, decodedOutput string) error {
	fmt.Println("zeroGTFS Roundtrip: GTFS -> Binary -> GTFS")
	fmt.Println("==========================================")
	fmt.Println()

	// Step 1: Load GTFS data
	fmt.Printf("Step 1: Loading GTFS data from: %s\n", source)
	gtfsData, err := loadGTFSData(source)
	if err != nil {
		return fmt.Errorf("failed to load GTFS data: %w", err)
	}

	fmt.Printf("  ✓ Agencies: %d\n", len(gtfsData.Agencies))
	fmt.Printf("  ✓ Stops: %d\n", len(gtfsData.Stops))
	fmt.Printf("  ✓ Routes: %d\n", len(gtfsData.Routes))
	fmt.Printf("  ✓ Trips: %d\n", len(gtfsData.Trips))
	fmt.Printf("  ✓ Stop Times: %d\n", len(gtfsData.StopTimes))

	originalSize := EstimateCSVSize(gtfsData)
	fmt.Printf("  Estimated original CSV size: ~%d bytes\n", originalSize)

	// Step 2: Encode to zeroGTFS binary format
	fmt.Println("\nStep 2: Encoding to zeroGTFS binary format...")
	encoder := NewZeroGTFSEncoder(gtfsData)
	if err := encoder.Encode(encodedOutput); err != nil {
		return fmt.Errorf("failed to encode: %w", err)
	}

	encodedFileInfo, err := os.Stat(encodedOutput)
	if err != nil {
		return fmt.Errorf("failed to stat encoded file: %w", err)
	}
	encodedSize := encodedFileInfo.Size()

	fmt.Printf("  ✓ Encoded file: %s (%d bytes)\n", encodedOutput, encodedSize)
	ratio := float64(originalSize) / float64(encodedSize)
	compression := ((float64(originalSize) - float64(encodedSize)) / float64(originalSize)) * 100
	fmt.Printf("  Compression ratio: %.2fx | Space saved: %.1f%%\n", ratio, compression)

	// Step 3: Decode from zeroGTFS binary format
	fmt.Println("\nStep 3: Decoding from zeroGTFS binary format...")
	decodedFeed, err := DecodeZeroGTFSFromFile(encodedOutput)
	if err != nil {
		return fmt.Errorf("failed to decode: %w", err)
	}
	fmt.Println("  ✓ Successfully decoded")

	// Step 4: Verify data integrity
	fmt.Println("\nStep 4: Verifying data integrity...")
	if len(decodedFeed.Agencies) != len(gtfsData.Agencies) {
		fmt.Printf("  ⚠ Agency count mismatch: %d vs %d\n", len(decodedFeed.Agencies), len(gtfsData.Agencies))
	} else {
		fmt.Printf("  ✓ Agencies: %d (match)\n", len(decodedFeed.Agencies))
	}

	if len(decodedFeed.Stops) != len(gtfsData.Stops) {
		fmt.Printf("  ⚠ Stop count mismatch: %d vs %d\n", len(decodedFeed.Stops), len(gtfsData.Stops))
	} else {
		fmt.Printf("  ✓ Stops: %d (match)\n", len(decodedFeed.Stops))
	}

	if len(decodedFeed.Routes) != len(gtfsData.Routes) {
		fmt.Printf("  ⚠ Route count mismatch: %d vs %d\n", len(decodedFeed.Routes), len(gtfsData.Routes))
	} else {
		fmt.Printf("  ✓ Routes: %d (match)\n", len(decodedFeed.Routes))
	}

	if len(decodedFeed.Patterns) > 0 {
		fmt.Printf("  ✓ Trip Patterns: %d (clustered from %d trips)\n", len(decodedFeed.Patterns), len(gtfsData.Trips))
	}

	if len(decodedFeed.Schedules) > 0 {
		fmt.Printf("  ✓ Schedules: %d\n", len(decodedFeed.Schedules))
	}

	// Step 5: Export decoded data back to GTFS ZIP format
	fmt.Printf("\nStep 5: Exporting decoded data to GTFS format...\n")
	if err := exportGTFSZip(decodedFeed, decodedOutput); err != nil {
		return fmt.Errorf("failed to export to GTFS: %w", err)
	}

	decodedFileInfo, err := os.Stat(decodedOutput)
	if err != nil {
		return fmt.Errorf("failed to stat decoded output file: %w", err)
	}

	fmt.Printf("  ✓ Exported GTFS: %s (%d bytes)\n", decodedOutput, decodedFileInfo.Size())

	// Summary
	fmt.Println("\n" + "==========================================")
	fmt.Println("Roundtrip Summary:")
	fmt.Printf("  Original GTFS size:    ~%d bytes\n", originalSize)
	fmt.Printf("  Compressed size:       %d bytes (%.1f%% of original)\n", encodedSize, float64(encodedSize)*100/float64(originalSize))
	fmt.Printf("  Exported GTFS size:    %d bytes\n", decodedFileInfo.Size())
	fmt.Printf("  Compression ratio:     %.2fx\n", ratio)
	fmt.Println("  ✓ Roundtrip complete!")

	return nil
}
