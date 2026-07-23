package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type DecodedZeroGTFSFeed struct {
	Header      *ZeroGTFSFileHeader
	StringTable *StringTable
	Agencies    []*TransitAgency
	Stops       []*TransitStop
	Routes      []*TransitRoute
	Patterns    []*TripPattern
	Schedules   []*Schedule

	StopIndex   map[string]*TransitStop
	RouteIndex  map[string]*TransitRoute
	AgencyIndex map[string]*TransitAgency
}

type TransitAgency struct {
	AgencyID string
	Name     string
	URL      string
	Timezone string
	Lang     string
	Phone    string
	FareURL  string
}

type TransitStop struct {
	StopID        string
	Name          string
	Lat           float64
	Lon           float64
	Code          string
	Desc          string
	ZoneID        string
	URL           string
	LocationType  int
	ParentStation string
}

type TransitRoute struct {
	RouteID   string
	AgencyID  string
	ShortName string
	LongName  string
	Desc      string
	Type      int
	URL       string
	Color     uint32
	TextColor uint32
	SortOrder int32
}

func DecodeZeroGTFSFromFile(filename string) (*DecodedZeroGTFSFeed, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	reader := bytes.NewReader(data)

	header := &ZeroGTFSFileHeader{}
	if err := binary.Read(reader, binary.LittleEndian, header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	if header.Magic != ZeroGTFSMagic {
		return nil, fmt.Errorf("invalid magic number: %x (expected %x)", header.Magic, ZeroGTFSMagic)
	}

	result := &DecodedZeroGTFSFeed{
		Header:      header,
		StopIndex:   make(map[string]*TransitStop),
		RouteIndex:  make(map[string]*TransitRoute),
		AgencyIndex: make(map[string]*TransitAgency),
	}

	_, err = reader.Seek(int64(header.StringTableOff), 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to string table: %w", err)
	}
	result.StringTable = NewStringTable()
	if err := result.StringTable.ReadBinary(reader); err != nil {
		return nil, fmt.Errorf("failed to read string table: %w", err)
	}

	if err := readEncodedAgencies(reader, result); err != nil {
		return nil, fmt.Errorf("failed to read agencies: %w", err)
	}

	if err := readEncodedStops(reader, result); err != nil {
		return nil, fmt.Errorf("failed to read stops: %w", err)
	}

	if err := readEncodedRoutes(reader, result); err != nil {
		return nil, fmt.Errorf("failed to read routes: %w", err)
	}

	if err := readEncodedPatterns(reader, result); err != nil {
		return nil, fmt.Errorf("failed to read patterns: %w", err)
	}

	if err := readEncodedSchedules(reader, result); err != nil {
		return nil, fmt.Errorf("failed to read schedules: %w", err)
	}

	return result, nil
}

func readEncodedAgencies(reader io.ReadSeeker, result *DecodedZeroGTFSFeed) error {
	_, err := reader.Seek(int64(result.Header.AgencyOff), 0)
	if err != nil {
		return err
	}

	result.Agencies = make([]*TransitAgency, result.Header.AgencyCount)

	for i := uint32(0); i < result.Header.AgencyCount; i++ {
		enc := &EncodedAgency{}
		if err := binary.Read(reader, binary.LittleEndian, enc); err != nil {
			return err
		}

		// Read AgencyID string
		agencyIDBytes := make([]byte, enc.AgencyIDLen)
		if _, err := reader.Read(agencyIDBytes); err != nil {
			return err
		}

		agency := &TransitAgency{
			AgencyID: string(agencyIDBytes),
			Name:     result.StringTable.Lookup(enc.NameToken),
			URL:      result.StringTable.Lookup(enc.URLToken),
			Timezone: result.StringTable.Lookup(enc.TimezoneToken),
			Lang:     result.StringTable.Lookup(enc.LangToken),
			Phone:    result.StringTable.Lookup(enc.PhoneToken),
			FareURL:  result.StringTable.Lookup(enc.FareURLToken),
		}
		result.Agencies[i] = agency
		result.AgencyIndex[agency.AgencyID] = agency
	}

	return nil
}

func readEncodedStops(reader io.ReadSeeker, result *DecodedZeroGTFSFeed) error {
	_, err := reader.Seek(int64(result.Header.StopsOff), 0)
	if err != nil {
		return err
	}

	result.Stops = make([]*TransitStop, result.Header.StopsCount)

	for i := uint32(0); i < result.Header.StopsCount; i++ {
		enc := &EncodedStop{}
		if err := binary.Read(reader, binary.LittleEndian, enc); err != nil {
			return err
		}

		// Read StopID string
		stopIDBytes := make([]byte, enc.StopIDLen)
		if _, err := reader.Read(stopIDBytes); err != nil {
			return err
		}

		// Read ParentStation ID string (if exists)
		parentStation := ""
		if enc.ParentIDLen > 0 {
			parentBytes := make([]byte, enc.ParentIDLen)
			if _, err := reader.Read(parentBytes); err != nil {
				return err
			}
			parentStation = string(parentBytes)
		}

		stop := &TransitStop{
			StopID:        string(stopIDBytes),
			Name:          result.StringTable.Lookup(enc.NameToken),
			Lat:           float64(enc.Lat),
			Lon:           float64(enc.Lon),
			Code:          result.StringTable.Lookup(enc.CodeToken),
			Desc:          result.StringTable.Lookup(enc.DescToken),
			ZoneID:        result.StringTable.Lookup(enc.ZoneToken),
			URL:           result.StringTable.Lookup(enc.URLToken),
			LocationType:  int(enc.LocationType),
			ParentStation: parentStation,
		}
		result.Stops[i] = stop
		result.StopIndex[stop.StopID] = stop
	}

	return nil
}

func readEncodedRoutes(reader io.ReadSeeker, result *DecodedZeroGTFSFeed) error {
	_, err := reader.Seek(int64(result.Header.RoutesOff), 0)
	if err != nil {
		return err
	}

	result.Routes = make([]*TransitRoute, result.Header.RoutesCount)

	for i := uint32(0); i < result.Header.RoutesCount; i++ {
		enc := &EncodedRoute{}
		if err := binary.Read(reader, binary.LittleEndian, enc); err != nil {
			return err
		}

		// Read RouteID string
		routeIDBytes := make([]byte, enc.RouteIDLen)
		if _, err := reader.Read(routeIDBytes); err != nil {
			return err
		}

		// Read AgencyID string
		agencyIDBytes := make([]byte, enc.AgencyIDLen)
		if _, err := reader.Read(agencyIDBytes); err != nil {
			return err
		}

		route := &TransitRoute{
			RouteID:   string(routeIDBytes),
			AgencyID:  string(agencyIDBytes),
			ShortName: result.StringTable.Lookup(enc.ShortNameToken),
			LongName:  result.StringTable.Lookup(enc.LongNameToken),
			Desc:      result.StringTable.Lookup(enc.DescToken),
			Type:      int(enc.Type),
			URL:       result.StringTable.Lookup(enc.URLToken),
			Color:     enc.Color,
			TextColor: enc.TextColor,
			SortOrder: enc.SortOrder,
		}
		result.Routes[i] = route
		result.RouteIndex[route.RouteID] = route
	}

	return nil
}

func readEncodedPatterns(reader io.ReadSeeker, result *DecodedZeroGTFSFeed) error {
	_, err := reader.Seek(int64(result.Header.PatternsOff), 0)
	if err != nil {
		return err
	}

	result.Patterns = make([]*TripPattern, result.Header.PatternsCount)

	for i := uint32(0); i < result.Header.PatternsCount; i++ {
		enc := &EncodedTripPattern{}
		if err := binary.Read(reader, binary.LittleEndian, enc); err != nil {
			return err
		}

		stopTokens := make([]uint16, enc.StopCount)
		for j := 0; j < int(enc.StopCount); j++ {
			var token uint16
			if err := binary.Read(reader, binary.LittleEndian, &token); err != nil {
				return err
			}
			stopTokens[j] = token
		}

		deltaTimes := make([]uint16, enc.StopCount)
		for j := 0; j < int(enc.StopCount); j++ {
			var delta uint16
			if err := binary.Read(reader, binary.LittleEndian, &delta); err != nil {
				return err
			}
			deltaTimes[j] = delta
		}

		pattern := &TripPattern{
			PatternID:    enc.PatternID,
			StopCount:    enc.StopCount,
			StopIDTokens: stopTokens,
			DeltaTimes:   deltaTimes,
		}
		result.Patterns[i] = pattern
	}

	return nil
}

func readEncodedSchedules(reader io.ReadSeeker, result *DecodedZeroGTFSFeed) error {
	_, err := reader.Seek(int64(result.Header.SchedulesOff), 0)
	if err != nil {
		return err
	}

	result.Schedules = make([]*Schedule, result.Header.SchedulesCount)

	for i := uint32(0); i < result.Header.SchedulesCount; i++ {
		enc := &EncodedSchedule{}
		if err := binary.Read(reader, binary.LittleEndian, enc); err != nil {
			return err
		}

		schedule := &Schedule{
			PatternID:           enc.PatternID,
			ServiceIndex:        enc.ServiceIndex,
			RouteIndex:          enc.RouteIndex,
			HeadsignToken:       enc.HeadsignToken,
			FirstStopDepartTime: enc.FirstStopDepartTime,
			Flags:               enc.Flags,
		}
		result.Schedules[i] = schedule
	}

	return nil
}

func (d *DecodedZeroGTFSFeed) GetStopsByTrip(tripIndex uint32) ([]TripStopInfo, error) {
	// Get schedule by trip index
	if tripIndex >= uint32(len(d.Schedules)) {
		return nil, fmt.Errorf("invalid trip index %d", tripIndex)
	}

	schedule := d.Schedules[tripIndex]

	if schedule.PatternID >= uint32(len(d.Patterns)) {
		return nil, fmt.Errorf("invalid pattern ID %d", schedule.PatternID)
	}
	pattern := d.Patterns[schedule.PatternID]

	absoluteTimes := DeltaDecodeTimestamps(pattern.DeltaTimes)

	result := make([]TripStopInfo, pattern.StopCount)
	for i := 0; i < int(pattern.StopCount); i++ {
		// StopIDTokens now contains stop indices (not tokens)
		stopIndex := pattern.StopIDTokens[i]
		if stopIndex >= uint16(len(d.Stops)) {
			return nil, fmt.Errorf("invalid stop index %d", stopIndex)
		}
		stop := d.Stops[stopIndex]

		var arrivalTime uint32
		var departTime uint32

		if i == 0 {
			departTime = schedule.FirstStopDepartTime
			arrivalTime = departTime
		} else {
			arrivalTime = absoluteTimes[i]
			departTime = absoluteTimes[i]
		}

		result[i] = TripStopInfo{
			StopID:      stop.StopID,
			ArrivalTime: arrivalTime,
			DepartTime:  departTime,
			Sequence:    uint32(i),
		}
	}

	return result, nil
}

type TripStopInfo struct {
	StopID      string
	ArrivalTime uint32
	DepartTime  uint32
	Sequence    uint32
}
