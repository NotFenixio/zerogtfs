package zerogtfs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"sort"
	"strconv"
)

type ZeroGTFSEncoder struct {
	data          *MockGTFSData
	stringTable   *StringTable
	patterns      map[uint64]*TripPattern
	schedules     []*Schedule
	patternIndex  map[uint64]uint32
	nextPatternID uint32

	// Index maps for converting GTFS IDs to array indices
	stopIDToIndex    map[string]uint16
	routeIDToIndex   map[string]uint16
	serviceIDToIndex map[string]uint16
	agencyIDToIndex  map[string]uint16
}

func NewZeroGTFSEncoder(data *MockGTFSData) *ZeroGTFSEncoder {
	return &ZeroGTFSEncoder{
		data:         data,
		stringTable:  NewStringTable(),
		patterns:     make(map[uint64]*TripPattern),
		schedules:    make([]*Schedule, 0),
		patternIndex: make(map[uint64]uint32),

		// Initialize index maps
		stopIDToIndex:    make(map[string]uint16),
		routeIDToIndex:   make(map[string]uint16),
		serviceIDToIndex: make(map[string]uint16),
		agencyIDToIndex:  make(map[string]uint16),
	}
}

func (e *ZeroGTFSEncoder) Encode(filename string) error {

	// Build index maps for all GTFS entities
	e.buildIndexMaps()

	// Tokenize only display strings (not internal IDs)
	e.tokenizeAllStrings()

	// Extract trip patterns with stop indices and fixed delta times
	e.extractTripPatterns()

	// Build schedules with indices instead of tokens
	e.buildSchedules()

	return e.writeBinaryFile(filename)
}

// buildIndexMaps creates mappings from GTFS IDs to 0-based array indices
// This allows us to avoid storing full strings for internal IDs, using compact uint16 indices instead
func (e *ZeroGTFSEncoder) buildIndexMaps() {

	// Map stop IDs to their array indices
	for i, stop := range e.data.Stops {
		e.stopIDToIndex[stop.StopId] = uint16(i)
	}

	// Map route IDs to their array indices
	for i, route := range e.data.Routes {
		e.routeIDToIndex[route.RouteId] = uint16(i)
	}

	// Map service IDs to their array indices (note: need to extract unique service IDs from trips)
	serviceIDMap := make(map[string]bool)
	for _, trip := range e.data.Trips {
		serviceIDMap[trip.ServiceId] = true
	}
	serviceIndex := uint16(0)
	for serviceID := range serviceIDMap {
		e.serviceIDToIndex[serviceID] = serviceIndex
		serviceIndex++
	}

	// Map agency IDs to their array indices
	for i, agency := range e.data.Agencies {
		e.agencyIDToIndex[agency.AgencyId] = uint16(i)
	}

	fmt.Printf("    - Built maps: %d stops, %d routes, %d services, %d agencies\n",
		len(e.stopIDToIndex), len(e.routeIDToIndex), len(e.serviceIDToIndex), len(e.agencyIDToIndex))
}

// tokenizeAllStrings only tokenizes human-readable display strings (NOT internal IDs)
// This prevents the uint16 token overflow hazard by excluding IDs like stop_id, route_id, etc.
func (e *ZeroGTFSEncoder) tokenizeAllStrings() {

	// Tokenize ONLY display names/phone numbers from agencies (NOT AgencyID)
	for _, agency := range e.data.Agencies {
		e.stringTable.Tokenize(agency.Name)
		e.stringTable.Tokenize(agency.Phone)
		// NOTE: Do NOT tokenize agency.AgencyId - it's an internal ID
	}

	// Tokenize ONLY display info from stops (NOT StopID)
	for _, stop := range e.data.Stops {
		e.stringTable.Tokenize(stop.Name)
		e.stringTable.Tokenize(stop.Code)
		// NOTE: Do NOT tokenize stop.StopId - it's an internal ID
	}

	// Tokenize ONLY display names from routes (NOT RouteID or AgencyID)
	for _, route := range e.data.Routes {
		e.stringTable.Tokenize(route.ShortName)
		e.stringTable.Tokenize(route.LongName)
		// NOTE: Do NOT tokenize route.RouteId or route.AgencyId - they are internal IDs
	}

	// Tokenize ONLY headsigns from trips (NOT TripID, RouteID, or ServiceID)
	for _, trip := range e.data.Trips {
		e.stringTable.Tokenize(trip.Headsign)
		// NOTE: Do NOT tokenize trip.TripId, trip.RouteId, or trip.ServiceId - they are internal IDs
	}

	fmt.Printf("    - Total unique display strings: %d\n", len(e.stringTable.IDToString))
}

func (e *ZeroGTFSEncoder) extractTripPatterns() {

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

			stopIndices := make([]uint16, len(stopList))
			deltaTimes := make([]uint16, len(stopList))

			for i, st := range stopList {
				// Store 0-based stop index instead of string token
				stopIdx, ok := e.stopIDToIndex[st.StopId]
				if !ok {
					log.Printf("Warning: stop %s not found in index\n", st.StopId)
					continue
				}
				stopIndices[i] = stopIdx

				if i == 0 {
					// FIX: First stop delta time must be 0 (absolute time is stored separately in EncodedSchedule.FirstStopDepartTime)
					deltaTimes[i] = 0
				} else {
					// Calculate delta from previous stop
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
				StopIDTokens: stopIndices, // Now contains indices, not tokens
				DeltaTimes:   deltaTimes,
			}
			e.patternIndex[signature] = patternID
		}
	}

	fmt.Printf("    - Extracted %d unique patterns from %d trips\n", len(e.patterns), len(e.data.Trips))
}

func (e *ZeroGTFSEncoder) buildPatternSignature(stopList []*MockStopTime) uint64 {
	h := fnv.New64a()
	var buf [32]byte // Buffer reutilizable para no alocar memoria con strconv

	for i, st := range stopList {
		_, _ = h.Write([]byte(st.StopId))

		if i < len(stopList)-1 {
			delta := stopList[i+1].DepartTime - st.DepartTime
			_, _ = h.Write([]byte(":"))

			// Convertimos el número a bytes sin alocar memoria en el Heap
			b := strconv.AppendUint(buf[:0], uint64(delta), 10)
			_, _ = h.Write(b)
		}
	}
	return h.Sum64()
}

func (e *ZeroGTFSEncoder) buildSchedules() {

	// Cambiamos el mapa para usar el Hash numérico (uint64) en vez de strings pesados
	tripToPattern := make(map[string]uint64)

	// OPTIMIZACIÓN CRUCIAL: Agrupamos directamente usando los PUNTEROS existentes.
	// No crees un objeto nuevo con '&MockStopTime{...}', reutiliza el que ya tienes.
	tripStopTimes := make(map[string][]*MockStopTime)
	for _, st := range e.data.StopTimes {
		tripStopTimes[st.TripId] = append(tripStopTimes[st.TripId], st)
	}

	for tripID, stopList := range tripStopTimes {
		sort.Slice(stopList, func(i, j int) bool {
			return stopList[i].StopSequence < stopList[j].StopSequence
		})
		// Ahora devuelve un uint64
		sig := e.buildPatternSignature(stopList)
		tripToPattern[tripID] = sig
	}

	for _, trip := range e.data.Trips {
		hash, ok := tripToPattern[trip.TripId]
		if !ok {
			log.Printf("Warning: trip %s has no stop times\n", trip.TripId)
			continue
		}

		// NOTA: Recuerda cambiar el tipo de tu mapa 'e.patternIndex'
		// para que sea map[uint64]int en lugar de map[string]int
		patternID := e.patternIndex[hash]
		stopList := tripStopTimes[trip.TripId]

		serviceIdx, ok := e.serviceIDToIndex[trip.ServiceId]
		if !ok {
			log.Printf("Warning: service %s not found in index\n", trip.ServiceId)
			continue
		}

		routeIdx, ok := e.routeIDToIndex[trip.RouteId]
		if !ok {
			log.Printf("Warning: route %s not found in index\n", trip.RouteId)
			continue
		}

		wheelchairBit := uint8(0)
		if trip.WheelchairAccessible == "1" {
			wheelchairBit = 1
		}
		flags := PackScheduleFlags(uint8(trip.DirectionId), wheelchairBit)

		schedule := &Schedule{
			PatternID:           patternID,
			ServiceIndex:        serviceIdx,
			RouteIndex:          routeIdx,
			HeadsignToken:       e.stringTable.Tokenize(trip.Headsign),
			FirstStopDepartTime: stopList[0].DepartTime,
			Flags:               flags,
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
			AgencyIDLen:   uint16(len(agency.AgencyId)),
			NameToken:     e.stringTable.Tokenize(agency.Name),
			URLToken:      e.stringTable.Tokenize(agency.Url),
			TimezoneToken: e.stringTable.Tokenize(agency.Timezone),
			LangToken:     e.stringTable.Tokenize(agency.Lang),
			PhoneToken:    e.stringTable.Tokenize(agency.Phone),
			FareURLToken:  e.stringTable.Tokenize(agency.FareUrl),
		}
		binary.Write(buf, binary.LittleEndian, enc)
		buf.WriteString(agency.AgencyId)
	}
	header.AgencyCount = uint32(len(e.data.Agencies))

	header.StopsOff = uint32(buf.Len())
	for _, stop := range e.data.Stops {
		parentLen := uint8(0)
		if stop.ParentStation != "" {
			parentLen = uint8(len(stop.ParentStation))
		}

		enc := &EncodedStop{
			StopIDLen:    uint16(len(stop.StopId)),
			NameToken:    e.stringTable.Tokenize(stop.Name),
			Lat:          float32(stop.Lat),
			Lon:          float32(stop.Lon),
			CodeToken:    e.stringTable.Tokenize(stop.Code),
			DescToken:    e.stringTable.Tokenize(stop.Desc),
			ZoneToken:    e.stringTable.Tokenize(stop.ZoneId),
			URLToken:     e.stringTable.Tokenize(stop.Url),
			LocationType: uint8(stop.LocationType),
			ParentIDLen:  parentLen,
		}
		binary.Write(buf, binary.LittleEndian, enc)
		buf.WriteString(stop.StopId)
		if parentLen > 0 {
			buf.WriteString(stop.ParentStation)
		}
	}
	header.StopsCount = uint32(len(e.data.Stops))

	header.RoutesOff = uint32(buf.Len())
	for _, route := range e.data.Routes {
		enc := &EncodedRoute{
			RouteIDLen:     uint16(len(route.RouteId)),
			AgencyIDLen:    uint16(len(route.AgencyId)),
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
		buf.WriteString(route.RouteId)
		buf.WriteString(route.AgencyId)
	}
	header.RoutesCount = uint32(len(e.data.Routes))

	header.PatternsOff = uint32(buf.Len())
	for _, pattern := range e.patterns {
		enc := &EncodedTripPattern{
			PatternID: pattern.PatternID,
			StopCount: pattern.StopCount,
		}
		binary.Write(buf, binary.LittleEndian, enc)

		// Write stop indices (not tokens)
		for _, idx := range pattern.StopIDTokens {
			binary.Write(buf, binary.LittleEndian, idx)
		}

		// Write delta times
		for _, delta := range pattern.DeltaTimes {
			binary.Write(buf, binary.LittleEndian, delta)
		}
	}
	header.PatternsCount = uint32(len(e.patterns))

	header.SchedulesOff = uint32(buf.Len())
	for _, schedule := range e.schedules {
		enc := &EncodedSchedule{
			PatternID:           schedule.PatternID,
			ServiceIndex:        schedule.ServiceIndex,
			RouteIndex:          schedule.RouteIndex,
			HeadsignToken:       schedule.HeadsignToken,
			FirstStopDepartTime: schedule.FirstStopDepartTime,
			Flags:               schedule.Flags,
		}
		binary.Write(buf, binary.LittleEndian, enc)
	}
	header.SchedulesCount = uint32(len(e.schedules))

	headerBuf := &bytes.Buffer{}
	binary.Write(headerBuf, binary.LittleEndian, header)
	copy(buf.Bytes()[headerPos:], headerBuf.Bytes())

	return os.WriteFile(filename, buf.Bytes(), 0644)
}
