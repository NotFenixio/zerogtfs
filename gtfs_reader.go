package main

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func ReadGTFSFromZip(zipPath string) (*MockGTFSData, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer reader.Close()

	data := &MockGTFSData{
		Agencies:  make([]*MockAgency, 0),
		Stops:     make([]*MockStop, 0),
		Routes:    make([]*MockRoute, 0),
		Trips:     make([]*MockTrip, 0),
		StopTimes: make([]*MockStopTime, 0),
	}

	fileMap := make(map[string]*zip.File)
	for _, file := range reader.File {
		fileMap[file.Name] = file
	}

	if agenciesFile, ok := fileMap["agency.txt"]; ok {
		if err := readAgencies(agenciesFile, data); err != nil {
			return nil, err
		}
	}

	if stopsFile, ok := fileMap["stops.txt"]; ok {
		if err := readStops(stopsFile, data); err != nil {
			return nil, err
		}
	}

	if routesFile, ok := fileMap["routes.txt"]; ok {
		if err := readRoutes(routesFile, data); err != nil {
			return nil, err
		}
	}

	if tripsFile, ok := fileMap["trips.txt"]; ok {
		if err := readTrips(tripsFile, data); err != nil {
			return nil, err
		}
	}

	if stopTimesFile, ok := fileMap["stop_times.txt"]; ok {
		if err := readStopTimes(stopTimesFile, data); err != nil {
			return nil, err
		}
	}

	return data, nil
}

func DownloadFile(url string, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: HTTP status %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func readAgencies(agenciesFile *zip.File, data *MockGTFSData) error {
	rc, err := agenciesFile.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	reader := csv.NewReader(rc)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read agencies header: %w", err)
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[h] = i
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading agencies record: %w", err)
		}

		agency := &MockAgency{}

		if idx, ok := headerMap["agency_id"]; ok && idx < len(record) {
			agency.AgencyId = record[idx]
		}
		if idx, ok := headerMap["agency_name"]; ok && idx < len(record) {
			agency.Name = record[idx]
		}
		if idx, ok := headerMap["agency_url"]; ok && idx < len(record) {
			agency.Url = record[idx]
		}
		if idx, ok := headerMap["agency_timezone"]; ok && idx < len(record) {
			agency.Timezone = record[idx]
		}
		if idx, ok := headerMap["agency_lang"]; ok && idx < len(record) {
			agency.Lang = record[idx]
		}
		if idx, ok := headerMap["agency_phone"]; ok && idx < len(record) {
			agency.Phone = record[idx]
		}
		if idx, ok := headerMap["agency_fare_url"]; ok && idx < len(record) {
			agency.FareUrl = record[idx]
		}

		data.Agencies = append(data.Agencies, agency)
	}

	log.Printf("Read %d agencies", len(data.Agencies))
	return nil
}

func readStops(stopsFile *zip.File, data *MockGTFSData) error {
	rc, err := stopsFile.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	reader := csv.NewReader(rc)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read stops header: %w", err)
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[h] = i
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading stops record: %w", err)
		}

		stop := &MockStop{}

		if idx, ok := headerMap["stop_id"]; ok && idx < len(record) {
			stop.StopId = record[idx]
		}
		if idx, ok := headerMap["stop_name"]; ok && idx < len(record) {
			stop.Name = record[idx]
		}
		if idx, ok := headerMap["stop_lat"]; ok && idx < len(record) {
			lat, err := strconv.ParseFloat(record[idx], 64)
			if err == nil {
				stop.Lat = lat
			}
		}
		if idx, ok := headerMap["stop_lon"]; ok && idx < len(record) {
			lon, err := strconv.ParseFloat(record[idx], 64)
			if err == nil {
				stop.Lon = lon
			}
		}
		if idx, ok := headerMap["stop_code"]; ok && idx < len(record) {
			stop.Code = record[idx]
		}
		if idx, ok := headerMap["stop_desc"]; ok && idx < len(record) {
			stop.Desc = record[idx]
		}
		if idx, ok := headerMap["zone_id"]; ok && idx < len(record) {
			stop.ZoneId = record[idx]
		}
		if idx, ok := headerMap["stop_url"]; ok && idx < len(record) {
			stop.Url = record[idx]
		}
		if idx, ok := headerMap["location_type"]; ok && idx < len(record) {
			locType, err := strconv.ParseInt(record[idx], 10, 32)
			if err == nil {
				stop.LocationType = int32(locType)
			}
		}
		if idx, ok := headerMap["parent_station"]; ok && idx < len(record) {
			stop.ParentStation = record[idx]
		}

		data.Stops = append(data.Stops, stop)
	}

	log.Printf("Read %d stops", len(data.Stops))
	return nil
}

func readRoutes(routesFile *zip.File, data *MockGTFSData) error {
	rc, err := routesFile.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	reader := csv.NewReader(rc)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read routes header: %w", err)
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[h] = i
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading routes record: %w", err)
		}

		route := &MockRoute{}

		if idx, ok := headerMap["route_id"]; ok && idx < len(record) {
			route.RouteId = record[idx]
		}
		if idx, ok := headerMap["agency_id"]; ok && idx < len(record) {
			route.AgencyId = record[idx]
		}
		if idx, ok := headerMap["route_short_name"]; ok && idx < len(record) {
			route.ShortName = record[idx]
		}
		if idx, ok := headerMap["route_long_name"]; ok && idx < len(record) {
			route.LongName = record[idx]
		}
		if idx, ok := headerMap["route_desc"]; ok && idx < len(record) {
			route.Desc = record[idx]
		}
		if idx, ok := headerMap["route_type"]; ok && idx < len(record) {
			routeType, err := strconv.ParseInt(record[idx], 10, 32)
			if err == nil {
				route.Type = int32(routeType)
			}
		}
		if idx, ok := headerMap["route_url"]; ok && idx < len(record) {
			route.Url = record[idx]
		}
		if idx, ok := headerMap["route_color"]; ok && idx < len(record) && record[idx] != "" {

			colorStr := strings.TrimPrefix(record[idx], "#")
			if len(colorStr) == 6 {
				color, err := strconv.ParseUint(colorStr, 16, 32)
				if err == nil {
					route.Color = uint32(color)
				}
			}
		}
		if idx, ok := headerMap["route_text_color"]; ok && idx < len(record) && record[idx] != "" {
			textColorStr := strings.TrimPrefix(record[idx], "#")
			if len(textColorStr) == 6 {
				textColor, err := strconv.ParseUint(textColorStr, 16, 32)
				if err == nil {
					route.TextColor = uint32(textColor)
				}
			}
		}
		if idx, ok := headerMap["route_sort_order"]; ok && idx < len(record) {
			sortOrder, err := strconv.ParseInt(record[idx], 10, 32)
			if err == nil {
				route.SortOrder = int32(sortOrder)
			}
		}

		data.Routes = append(data.Routes, route)
	}

	log.Printf("Read %d routes", len(data.Routes))
	return nil
}

func readTrips(tripsFile *zip.File, data *MockGTFSData) error {
	rc, err := tripsFile.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	reader := csv.NewReader(rc)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read trips header: %w", err)
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[h] = i
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading trips record: %w", err)
		}

		trip := &MockTrip{}

		if idx, ok := headerMap["trip_id"]; ok && idx < len(record) {
			trip.TripId = record[idx]
		}
		if idx, ok := headerMap["route_id"]; ok && idx < len(record) {
			trip.RouteId = record[idx]
		}
		if idx, ok := headerMap["service_id"]; ok && idx < len(record) {
			trip.ServiceId = record[idx]
		}
		if idx, ok := headerMap["shape_id"]; ok && idx < len(record) {
			trip.ShapeId = record[idx]
		}
		if idx, ok := headerMap["direction_id"]; ok && idx < len(record) {
			directionID, err := strconv.ParseInt(record[idx], 10, 32)
			if err == nil {
				trip.DirectionId = int32(directionID)
			}
		}
		if idx, ok := headerMap["trip_headsign"]; ok && idx < len(record) {
			trip.Headsign = record[idx]
		}
		if idx, ok := headerMap["block_id"]; ok && idx < len(record) {
			blockID, err := strconv.ParseInt(record[idx], 10, 32)
			if err == nil {
				trip.BlockId = int32(blockID)
			}
		}
		if idx, ok := headerMap["wheelchair_accessible"]; ok && idx < len(record) {
			trip.WheelchairAccessible = record[idx]
		}

		data.Trips = append(data.Trips, trip)
	}

	log.Printf("Read %d trips", len(data.Trips))
	return nil
}

func readStopTimes(stopTimesFile *zip.File, data *MockGTFSData) error {
	rc, err := stopTimesFile.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	reader := csv.NewReader(rc)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read stop_times header: %w", err)
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[h] = i
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading stop_times record: %w", err)
		}

		stopTime := &MockStopTime{}

		if idx, ok := headerMap["trip_id"]; ok && idx < len(record) {
			stopTime.TripId = record[idx]
		}
		if idx, ok := headerMap["stop_id"]; ok && idx < len(record) {
			stopTime.StopId = record[idx]
		}
		if idx, ok := headerMap["stop_sequence"]; ok && idx < len(record) {
			seq, err := strconv.ParseInt(record[idx], 10, 32)
			if err == nil {
				stopTime.StopSequence = uint32(seq)
			}
		}
		if idx, ok := headerMap["arrival_time"]; ok && idx < len(record) && record[idx] != "" {
			stopTime.ArrivalTime = TimeToSeconds(record[idx])
		}
		if idx, ok := headerMap["departure_time"]; ok && idx < len(record) && record[idx] != "" {
			stopTime.DepartTime = TimeToSeconds(record[idx])
		}
		if idx, ok := headerMap["pickup_type"]; ok && idx < len(record) {
			pickupType, err := strconv.ParseInt(record[idx], 10, 32)
			if err == nil {
				stopTime.PickupType = int32(pickupType)
			}
		}
		if idx, ok := headerMap["drop_off_type"]; ok && idx < len(record) {
			dropOffType, err := strconv.ParseInt(record[idx], 10, 32)
			if err == nil {
				stopTime.DropOffType = int32(dropOffType)
			}
		}

		data.StopTimes = append(data.StopTimes, stopTime)
	}

	log.Printf("Read %d stop times", len(data.StopTimes))
	return nil
}
