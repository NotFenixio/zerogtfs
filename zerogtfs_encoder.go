package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"sort"
)

type ZeroGTFSEncoder struct {
	data          *MockGTFSData
	stringTable   *StringTable
	patterns      map[string]*TripPattern
	schedules     []*Schedule
	patternIndex  map[string]uint32
	nextPatternID uint32
}

func NewZeroGTFSEncoder(data *MockGTFSData) *ZeroGTFSEncoder {
	return &ZeroGTFSEncoder{
		data:         data,
		stringTable:  NewStringTable(),
		patterns:     make(map[string]*TripPattern),
		schedules:    make([]*Schedule, 0),
		patternIndex: make(map[string]uint32),
	}
}

func (e *ZeroGTFSEncoder) Encode(filename string) error {

	e.tokenizeAllStrings()

	e.extractTripPatterns()

	e.buildSchedules()

	return e.writeBinaryFile(filename)
}

func (e *ZeroGTFSEncoder) tokenizeAllStrings() {
	fmt.Println("    - Tokenizing strings...")

	for _, agency := range e.data.Agencies {
		e.stringTable.Tokenize(agency.AgencyId)
		e.stringTable.Tokenize(agency.Name)
		e.stringTable.Tokenize(agency.Url)
		e.stringTable.Tokenize(agency.Timezone)
		e.stringTable.Tokenize(agency.Lang)
		e.stringTable.Tokenize(agency.Phone)
		e.stringTable.Tokenize(agency.FareUrl)
	}

	for _, stop := range e.data.Stops {
		e.stringTable.Tokenize(stop.StopId)
		e.stringTable.Tokenize(stop.Name)
		e.stringTable.Tokenize(stop.Code)
		e.stringTable.Tokenize(stop.Desc)
		e.stringTable.Tokenize(stop.ZoneId)
		e.stringTable.Tokenize(stop.Url)
		e.stringTable.Tokenize(stop.ParentStation)
	}

	for _, route := range e.data.Routes {
		e.stringTable.Tokenize(route.RouteId)
		e.stringTable.Tokenize(route.AgencyId)
		e.stringTable.Tokenize(route.ShortName)
		e.stringTable.Tokenize(route.LongName)
		e.stringTable.Tokenize(route.Desc)
		e.stringTable.Tokenize(route.Url)
	}

	for _, trip := range e.data.Trips {
		e.stringTable.Tokenize(trip.TripId)
		e.stringTable.Tokenize(trip.RouteId)
		e.stringTable.Tokenize(trip.ServiceId)
		e.stringTable.Tokenize(trip.ShapeId)
		e.stringTable.Tokenize(trip.Headsign)
	}

	for _, st := range e.data.StopTimes {
		e.stringTable.Tokenize(st.TripId)
		e.stringTable.Tokenize(st.StopId)
	}

	fmt.Printf("    - Total unique strings: %d\n", len(e.stringTable.IDToString))
}

func (e *ZeroGTFSEncoder) extractTripPatterns() {
	fmt.Println("    - Extracting trip patterns (clustering)...")

	tripStopTimes := make(map[string][]*MockStopTime)
	for _, st := range e.data.StopTimes {

		tripStopTimes[st.TripId] = append(tripStopTimes[st.TripId], &MockStopTime{
			StopId:       st.StopId,
			StopSequence: st.StopSequence,
			ArrivalTime:  st.ArrivalTime,
			DepartTime:   st.DepartTime,
		})
	}

	for _, stopList := range tripStopTimes {
		sort.Slice(stopList, func(i, j int) bool {
			return stopList[i].StopSequence < stopList[j].StopSequence
		})
	}

	for _, stopList := range tripStopTimes {
		if len(stopList) == 0 {
			continue
		}

		signature := e.buildPatternSignature(stopList)

		if _, exists := e.patterns[signature]; !exists {
			patternID := e.nextPatternID
			e.nextPatternID++

			stopTokens := make([]uint16, len(stopList))
			deltaTimes := make([]uint16, len(stopList))

			for i, st := range stopList {
				stopTokens[i] = e.stringTable.Tokenize(st.StopId)

				if i == 0 {

					deltaTimes[i] = uint16(st.DepartTime % 65536)
				} else {

					delta := st.DepartTime - stopList[i-1].DepartTime
					if delta > 65535 {
						deltaTimes[i] = 65535
					} else {
						deltaTimes[i] = uint16(delta)
					}
				}
			}

			e.patterns[signature] = &TripPattern{
				PatternID:    patternID,
				StopCount:    uint16(len(stopList)),
				StopIDTokens: stopTokens,
				DeltaTimes:   deltaTimes,
			}
			e.patternIndex[signature] = patternID
		}
	}

	fmt.Printf("    - Extracted %d unique patterns from %d trips\n", len(e.patterns), len(e.data.Trips))
}

func (e *ZeroGTFSEncoder) buildPatternSignature(stopList []*MockStopTime) string {
	sig := ""
	for i, st := range stopList {
		sig += st.StopId
		if i < len(stopList)-1 {
			delta := stopList[i+1].DepartTime - st.DepartTime
			sig += fmt.Sprintf(":%d", delta)
		}
	}
	return sig
}

func (e *ZeroGTFSEncoder) buildSchedules() {
	fmt.Println("    - Building schedules...")

	tripToPattern := make(map[string]string)

	tripStopTimes := make(map[string][]*MockStopTime)
	for _, st := range e.data.StopTimes {
		tripStopTimes[st.TripId] = append(tripStopTimes[st.TripId], &MockStopTime{
			StopId:       st.StopId,
			StopSequence: st.StopSequence,
			ArrivalTime:  st.ArrivalTime,
			DepartTime:   st.DepartTime,
		})
	}

	for tripID, stopList := range tripStopTimes {
		sort.Slice(stopList, func(i, j int) bool {
			return stopList[i].StopSequence < stopList[j].StopSequence
		})
		sig := e.buildPatternSignature(stopList)
		tripToPattern[tripID] = sig
	}

	for _, trip := range e.data.Trips {
		sig, ok := tripToPattern[trip.TripId]
		if !ok {
			log.Printf("Warning: trip %s has no stop times\n", trip.TripId)
			continue
		}

		patternID := e.patternIndex[sig]
		stopList := tripStopTimes[trip.TripId]

		wheelchairBit := uint8(0)
		if trip.WheelchairAccessible == "1" {
			wheelchairBit = 1
		}

		schedule := &Schedule{
			PatternID:            patternID,
			TripIDToken:          e.stringTable.Tokenize(trip.TripId),
			ServiceIDToken:       e.stringTable.Tokenize(trip.ServiceId),
			HeadsignToken:        e.stringTable.Tokenize(trip.Headsign),
			FirstStopDepartTime:  stopList[0].DepartTime,
			RouteIDToken:         e.stringTable.Tokenize(trip.RouteId),
			DirectionID:          uint8(trip.DirectionId),
			WheelchairAccessible: wheelchairBit,
		}
		e.schedules = append(e.schedules, schedule)
	}

	fmt.Printf("    - Created %d schedule entries\n", len(e.schedules))
}

func (e *ZeroGTFSEncoder) writeBinaryFile(filename string) error {
	fmt.Printf("    - Writing binary file: %s\n", filename)

	buf := &bytes.Buffer{}

	headerPos := buf.Len()
	header := &ZeroGTFSFileHeader{
		Magic:   ZeroGTFSMagic,
		Version: ZeroGTFSVersion,
	}
	binary.Write(buf, binary.LittleEndian, header)

	header.StringTableOff = uint32(buf.Len())
	e.stringTable.WriteBinary(buf)
	header.StringTableSize = uint32(buf.Len()) - header.StringTableOff

	header.AgencyOff = uint32(buf.Len())
	for _, agency := range e.data.Agencies {
		enc := &EncodedAgency{
			AgencyIDToken: e.stringTable.Tokenize(agency.AgencyId),
			NameToken:     e.stringTable.Tokenize(agency.Name),
			URLToken:      e.stringTable.Tokenize(agency.Url),
			TimezoneToken: e.stringTable.Tokenize(agency.Timezone),
			LangToken:     e.stringTable.Tokenize(agency.Lang),
			PhoneToken:    e.stringTable.Tokenize(agency.Phone),
			FareURLToken:  e.stringTable.Tokenize(agency.FareUrl),
		}
		binary.Write(buf, binary.LittleEndian, enc)
	}
	header.AgencyCount = uint32(len(e.data.Agencies))

	header.StopsOff = uint32(buf.Len())
	for _, stop := range e.data.Stops {
		enc := &EncodedStop{
			StopIDToken:  e.stringTable.Tokenize(stop.StopId),
			NameToken:    e.stringTable.Tokenize(stop.Name),
			Lat:          float32(stop.Lat),
			Lon:          float32(stop.Lon),
			CodeToken:    e.stringTable.Tokenize(stop.Code),
			DescToken:    e.stringTable.Tokenize(stop.Desc),
			ZoneToken:    e.stringTable.Tokenize(stop.ZoneId),
			URLToken:     e.stringTable.Tokenize(stop.Url),
			LocationType: uint8(stop.LocationType),
			ParentToken:  e.stringTable.Tokenize(stop.ParentStation),
		}
		binary.Write(buf, binary.LittleEndian, enc)
	}
	header.StopsCount = uint32(len(e.data.Stops))

	header.RoutesOff = uint32(buf.Len())
	for _, route := range e.data.Routes {
		enc := &EncodedRoute{
			RouteIDToken:   e.stringTable.Tokenize(route.RouteId),
			AgencyIDToken:  e.stringTable.Tokenize(route.AgencyId),
			ShortNameToken: e.stringTable.Tokenize(route.ShortName),
			LongNameToken:  e.stringTable.Tokenize(route.LongName),
			DescToken:      e.stringTable.Tokenize(route.Desc),
			Type:           uint8(route.Type),
			URLToken:       e.stringTable.Tokenize(route.Url),
			Color:          route.Color,
			TextColor:      route.TextColor,
			SortOrder:      route.SortOrder,
		}
		binary.Write(buf, binary.LittleEndian, enc)
	}
	header.RoutesCount = uint32(len(e.data.Routes))

	header.PatternsOff = uint32(buf.Len())
	for _, pattern := range e.patterns {
		enc := &EncodedTripPattern{
			PatternID: pattern.PatternID,
			StopCount: pattern.StopCount,
		}
		binary.Write(buf, binary.LittleEndian, enc)

		for _, token := range pattern.StopIDTokens {
			binary.Write(buf, binary.LittleEndian, token)
		}

		for _, delta := range pattern.DeltaTimes {
			binary.Write(buf, binary.LittleEndian, delta)
		}
	}
	header.PatternsCount = uint32(len(e.patterns))

	header.SchedulesOff = uint32(buf.Len())
	for _, schedule := range e.schedules {
		enc := &EncodedSchedule{
			PatternID:            schedule.PatternID,
			TripIDToken:          schedule.TripIDToken,
			ServiceIDToken:       schedule.ServiceIDToken,
			HeadsignToken:        schedule.HeadsignToken,
			FirstStopDepartTime:  schedule.FirstStopDepartTime,
			RouteIDToken:         schedule.RouteIDToken,
			DirectionID:          schedule.DirectionID,
			WheelchairAccessible: schedule.WheelchairAccessible,
		}
		binary.Write(buf, binary.LittleEndian, enc)
	}
	header.SchedulesCount = uint32(len(e.schedules))

	headerBuf := &bytes.Buffer{}
	binary.Write(headerBuf, binary.LittleEndian, header)
	copy(buf.Bytes()[headerPos:], headerBuf.Bytes())

	return os.WriteFile(filename, buf.Bytes(), 0644)
}
