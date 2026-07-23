package main

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ZeroGTFSFileHeader is the fixed-size header at the start of every zeroGTFS file
// All offsets are absolute byte positions from the start of the file.
type ZeroGTFSFileHeader struct {
	Magic           uint32 // 0x5a455250 = "ZEROP" in hex (for zeroGTFS)
	Version         uint16 // Format version
	Reserved        uint16 // Reserved for future use
	StringTableOff  uint32 // Offset to string dictionary
	StringTableSize uint32 // Size of string table in bytes
	AgencyOff       uint32 // Offset to agencies
	AgencyCount     uint32 // Number of agencies
	StopsOff        uint32 // Offset to stops
	StopsCount      uint32 // Number of stops
	RoutesOff       uint32 // Offset to routes
	RoutesCount     uint32 // Number of routes
	PatternsOff     uint32 // Offset to trip patterns
	PatternsCount   uint32 // Number of trip patterns
	SchedulesOff    uint32 // Offset to schedules (departure times for patterns)
	SchedulesCount  uint32 // Number of schedule entries
}

// EncodedStop is the binary layout for a transit stop
// Note: StopID is stored as a length-prefixed string (not tokenized) to avoid string table overflow
type EncodedStop struct {
	StopIDLen    uint16  // Length of StopID string
	NameToken    uint16  // Token ID for stop name (display string)
	Lat          float32 // Latitude (32-bit float for compression)
	Lon          float32 // Longitude (32-bit float)
	CodeToken    uint16  // Token for stop code (optional display string)
	DescToken    uint16  // Token for description (optional display string)
	ZoneToken    uint16  // Token for zone_id (optional display string)
	URLToken     uint16  // Token for URL (optional display string)
	LocationType uint8   // 0 = stop, 1 = station
	ParentIDLen  uint8   // Length of ParentStation ID (0 if none)
	Padding      uint8   // Padding for alignment
	// Followed by StopIDLen bytes of StopID string
	// Followed by ParentIDLen bytes of ParentStation ID string (if ParentIDLen > 0)
}

// EncodedAgency is the binary layout for a transit agency
// Note: AgencyID is stored as a length-prefixed string (not tokenized) to avoid string table overflow
type EncodedAgency struct {
	AgencyIDLen   uint16 // Length of AgencyID string
	NameToken     uint16 // Token for name (display string)
	URLToken      uint16 // Token for URL (optional display string)
	TimezoneToken uint16 // Token for timezone (display string)
	LangToken     uint16 // Token for language code (optional display string)
	PhoneToken    uint16 // Token for phone (display string)
	FareURLToken  uint16 // Token for fare_url (optional display string)
	Padding       uint8  // Padding for alignment
	// Followed by AgencyIDLen bytes of AgencyID string
}

// EncodedRoute is the binary layout for a transit route
// Note: RouteID and AgencyID are stored as length-prefixed strings (not tokenized) to avoid string table overflow
type EncodedRoute struct {
	RouteIDLen     uint16 // Length of RouteID string
	AgencyIDLen    uint16 // Length of AgencyID string
	ShortNameToken uint16 // Token for short_name (display string)
	LongNameToken  uint16 // Token for long_name (display string)
	DescToken      uint16 // Token for description (optional display string)
	Type           uint8  // Route type (0-7)
	Padding1       uint8  // Padding
	URLToken       uint16 // Token for URL (optional display string)
	Color          uint32 // RRGGBB color
	TextColor      uint32 // RRGGBB text color
	SortOrder      int32  // Sort order
	// Followed by RouteIDLen bytes of RouteID string
	// Followed by AgencyIDLen bytes of AgencyID string
}

// TripPattern represents a unique sequence of stops with time deltas
// This is the core of zeroGTFS compression
type TripPattern struct {
	PatternID    uint32   // Unique pattern ID (0-indexed)
	StopCount    uint16   // Number of stops in sequence
	StopIDTokens []uint16 // Token IDs for each stop (variable length)
	DeltaTimes   []uint16 // Delta times (seconds) between consecutive stops
}

// EncodedTripPattern is the binary layout for a trip pattern
type EncodedTripPattern struct {
	PatternID uint32 // Unique ID for this pattern
	StopCount uint16 // Number of stops
	Padding   uint16 // Padding for alignment
	// Followed by: StopCount * uint16 (stop token IDs)
	// Followed by: StopCount * uint16 (delta times in seconds)
}

// Schedule represents a single instance of a trip pattern with a departure time
type Schedule struct {
	PatternID           uint32 // Reference to trip pattern
	ServiceIndex        uint16 // 0-based index into services array
	RouteIndex          uint16 // 0-based index into routes array
	HeadsignToken       uint16 // Token for headsign (display string only)
	FirstStopDepartTime uint32 // Absolute seconds past midnight for first stop departure
	Flags               uint8  // Packed bitflags: DirectionID (1 bit) | WheelchairAccessible (2 bits) | reserved
}

// EncodedSchedule is the binary layout for a schedule
type EncodedSchedule struct {
	PatternID           uint32 // Pattern ID
	ServiceIndex        uint16 // 0-based index into services array
	RouteIndex          uint16 // 0-based index into routes array
	HeadsignToken       uint16 // Token for headsign (display string only)
	FirstStopDepartTime uint32 // Departure time for first stop
	Flags               uint8  // Packed bitflags: DirectionID (1 bit) | WheelchairAccessible (2 bits) | reserved
	Padding             uint8  // Padding for alignment
}

// StringTable maps strings to token IDs and vice versa
type StringTable struct {
	// token -> string mapping
	IDToString map[uint16]string
	// string -> token mapping
	StringToID map[string]uint16
	// Token counter
	NextToken uint16
}

// NewStringTable creates a new empty string table
func NewStringTable() *StringTable {
	return &StringTable{
		IDToString: make(map[uint16]string),
		StringToID: make(map[string]uint16),
		NextToken:  1, // Start at 1; 0 is reserved for empty/nil
	}
}

// Tokenize adds a string to the table and returns its token ID
// If the string is empty, returns 0 (reserved for nil/empty values)
func (st *StringTable) Tokenize(s string) uint16 {
	if s == "" {
		return 0
	}
	if token, exists := st.StringToID[s]; exists {
		return token
	}
	token := st.NextToken
	st.IDToString[token] = s
	st.StringToID[s] = token
	st.NextToken++
	return token
}

// Lookup retrieves a string by token ID
func (st *StringTable) Lookup(token uint16) string {
	if token == 0 {
		return ""
	}
	if s, exists := st.IDToString[token]; exists {
		return s
	}
	return ""
}

// WriteBinary writes the string table to binary format
// Format: [uint16 count][token1][len1][str1][token2][len2][str2]...
func (st *StringTable) WriteBinary(w io.Writer) error {
	// Write count of strings (excluding token 0)
	count := uint16(len(st.IDToString))
	if err := binary.Write(w, binary.LittleEndian, count); err != nil {
		return err
	}

	// Write each token-string pair
	for token := uint16(1); token < st.NextToken; token++ {
		s := st.IDToString[token]

		// Write token ID
		if err := binary.Write(w, binary.LittleEndian, token); err != nil {
			return err
		}

		// Write string length (uint16)
		if err := binary.Write(w, binary.LittleEndian, uint16(len(s))); err != nil {
			return err
		}

		// Write string bytes
		if _, err := w.Write([]byte(s)); err != nil {
			return err
		}
	}

	return nil
}

// ReadBinary reads a string table from binary format
func (st *StringTable) ReadBinary(r io.Reader) error {
	var count uint16
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return err
	}

	for i := uint16(0); i < count; i++ {
		var token uint16
		var strLen uint16

		if err := binary.Read(r, binary.LittleEndian, &token); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &strLen); err != nil {
			return err
		}

		strBytes := make([]byte, strLen)
		if _, err := r.Read(strBytes); err != nil {
			return err
		}

		s := string(strBytes)
		st.IDToString[token] = s
		st.StringToID[s] = token
		if token >= st.NextToken {
			st.NextToken = token + 1
		}
	}

	return nil
}

// DeltaEncodeTimestamps converts absolute timestamps to delta-encoded form
// Input: [1000, 1500, 1700, 2000] (absolute seconds)
// Output: [1000, 500, 200, 300] (first absolute, rest relative)
func DeltaEncodeTimestamps(times []uint32) []uint16 {
	if len(times) == 0 {
		return nil
	}

	deltas := make([]uint16, len(times))
	deltas[0] = uint16(times[0]) // First time is absolute (truncated to uint16 if needed)

	for i := 1; i < len(times); i++ {
		delta := times[i] - times[i-1]
		if delta > 65535 {
			fmt.Printf("Warning: delta time %d exceeds uint16 max, capping\n", delta)
			deltas[i] = 65535
		} else {
			deltas[i] = uint16(delta)
		}
	}

	return deltas
}

// DeltaDecodeTimestamps reconstructs absolute timestamps from delta-encoded form
// Input: [1000, 500, 200, 300] (first absolute, rest relative)
// Output: [1000, 1500, 1700, 2000] (absolute seconds)
func DeltaDecodeTimestamps(deltas []uint16) []uint32 {
	if len(deltas) == 0 {
		return nil
	}

	times := make([]uint32, len(deltas))
	times[0] = uint32(deltas[0])

	for i := 1; i < len(deltas); i++ {
		times[i] = times[i-1] + uint32(deltas[i])
	}

	return times
}

// The Magic constant for zeroGTFS files
const ZeroGTFSMagic uint32 = 0x5a455250 // "ZRPT" in ASCII (little-endian)
const ZeroGTFSVersion uint16 = 1

// PackScheduleFlags packs DirectionID (1 bit) and WheelchairAccessible (2 bits) into a single byte
// Bit layout: [reserved(5 bits) | WheelchairAccessible(2 bits) | DirectionID(1 bit)]
func PackScheduleFlags(directionID uint8, wheelchairAccessible uint8) uint8 {
	flags := uint8(0)
	flags |= (directionID & 0x01)                 // DirectionID in bit 0
	flags |= ((wheelchairAccessible & 0x03) << 1) // WheelchairAccessible in bits 1-2
	return flags
}

// UnpackScheduleFlags extracts DirectionID and WheelchairAccessible from packed flags
func UnpackScheduleFlags(flags uint8) (directionID uint8, wheelchairAccessible uint8) {
	directionID = flags & 0x01                 // Extract bit 0
	wheelchairAccessible = (flags >> 1) & 0x03 // Extract bits 1-2
	return
}

// UnpackStopIndexFromToken extracts the stop index from a uint16 token
// Note: In the refactored format, stop tokens are actually stop indices
func UnpackStopIndex(token uint16) uint16 {
	return token
}
