package main

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

func exportGTFSZip(feed *DecodedZeroGTFSFeed, outputFile string) error {
	zipFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Write agency.txt
	if err := writeAgencyCSV(zipWriter, feed); err != nil {
		zipWriter.Close()
		zipFile.Close()
		// Cleanup partial file
		if rmErr := os.Remove(outputFile); rmErr != nil {
			return fmt.Errorf("failed to write agency.txt: %w (cleanup error: %v)", err, rmErr)
		}
		return fmt.Errorf("failed to write agency.txt: %w", err)
	}

	// Write stops.txt
	if err := writeStopsCSV(zipWriter, feed); err != nil {
		zipWriter.Close()
		zipFile.Close()
		if rmErr := os.Remove(outputFile); rmErr != nil {
			return fmt.Errorf("failed to write stops.txt: %w (cleanup error: %v)", err, rmErr)
		}
		return fmt.Errorf("failed to write stops.txt: %w", err)
	}

	// Write routes.txt
	if err := writeRoutesCSV(zipWriter, feed); err != nil {
		zipWriter.Close()
		zipFile.Close()
		if rmErr := os.Remove(outputFile); rmErr != nil {
			return fmt.Errorf("failed to write routes.txt: %w (cleanup error: %v)", err, rmErr)
		}
		return fmt.Errorf("failed to write routes.txt: %w", err)
	}

	// Write trips.txt and stop_times.txt
	if err := writeTripsAndStopTimes(zipWriter, feed); err != nil {
		zipWriter.Close()
		zipFile.Close()
		if rmErr := os.Remove(outputFile); rmErr != nil {
			return fmt.Errorf("failed to write trips/stop_times: %w (cleanup error: %v)", err, rmErr)
		}
		return fmt.Errorf("failed to write trips/stop_times: %w", err)
	}

	return nil
}

func writeAgencyCSV(zipWriter *zip.Writer, feed *DecodedZeroGTFSFeed) error {
	w, err := zipWriter.Create("agency.txt")
	if err != nil {
		return err
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"agency_id", "agency_name", "agency_url", "agency_timezone", "agency_lang", "agency_phone", "agency_fare_url"})

	for _, agency := range feed.Agencies {
		writer.Write([]string{
			agency.AgencyID,
			agency.Name,
			agency.URL,
			agency.Timezone,
			agency.Lang,
			agency.Phone,
			agency.FareURL,
		})
	}

	return nil
}

func writeStopsCSV(zipWriter *zip.Writer, feed *DecodedZeroGTFSFeed) error {
	w, err := zipWriter.Create("stops.txt")
	if err != nil {
		return err
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"stop_id", "stop_name", "stop_lat", "stop_lon", "stop_code", "stop_desc", "zone_id", "stop_url", "location_type", "parent_station"})

	for _, stop := range feed.Stops {
		writer.Write([]string{
			stop.StopID,
			stop.Name,
			fmt.Sprintf("%.6f", stop.Lat),
			fmt.Sprintf("%.6f", stop.Lon),
			stop.Code,
			stop.Desc,
			stop.ZoneID,
			stop.URL,
			strconv.Itoa(stop.LocationType),
			stop.ParentStation,
		})
	}

	return nil
}

func writeRoutesCSV(zipWriter *zip.Writer, feed *DecodedZeroGTFSFeed) error {
	w, err := zipWriter.Create("routes.txt")
	if err != nil {
		return err
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"route_id", "agency_id", "route_short_name", "route_long_name", "route_desc", "route_type", "route_url", "route_color", "route_text_color", "route_sort_order"})

	for _, route := range feed.Routes {
		writer.Write([]string{
			route.RouteID,
			route.AgencyID,
			route.ShortName,
			route.LongName,
			route.Desc,
			strconv.Itoa(route.Type),
			route.URL,
			fmt.Sprintf("%06X", route.Color),
			fmt.Sprintf("%06X", route.TextColor),
			strconv.Itoa(int(route.SortOrder)),
		})
	}

	return nil
}

func fastTimeToString(buf []byte, seconds uint32) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	b := buf[:0]

	if hours < 10 {
		b = append(b, '0')
	}
	b = strconv.AppendUint(b, uint64(hours), 10)
	b = append(b, ':')

	if minutes < 10 {
		b = append(b, '0')
	}
	b = strconv.AppendUint(b, uint64(minutes), 10)
	b = append(b, ':')

	if secs < 10 {
		b = append(b, '0')
	}
	b = strconv.AppendUint(b, uint64(secs), 10)

	return string(b)
}

func writeTripsAndStopTimes(zipWriter *zip.Writer, feed *DecodedZeroGTFSFeed) error {
	tripsFile, err := zipWriter.Create("trips.txt")
	if err != nil {
		return err
	}

	stopTimesFile, err := zipWriter.Create("stop_times.txt")
	if err != nil {
		return err
	}

	tripsWriter := csv.NewWriter(tripsFile)
	defer tripsWriter.Flush()

	stopTimesWriter := csv.NewWriter(stopTimesFile)
	defer stopTimesWriter.Flush()

	tripsWriter.Write([]string{"trip_id", "route_id", "service_id", "trip_headsign"})
	stopTimesWriter.Write([]string{"trip_id", "arrival_time", "departure_time", "stop_id", "stop_sequence"})

	patternAbsoluteTimes := make([][]uint32, len(feed.Patterns))
	for i, pattern := range feed.Patterns {
		patternAbsoluteTimes[i] = DeltaDecodeTimestamps(pattern.DeltaTimes)
	}

	var bufTrip, bufService, bufTimeA, bufTimeD, bufSeq []byte

	for tripIdx, schedule := range feed.Schedules {
		bufTrip = append(bufTrip[:0], "trip_"...)
		bufTrip = strconv.AppendUint(bufTrip, uint64(tripIdx), 10)
		tripID := string(bufTrip)

		// Validate route index bounds
		if schedule.RouteIndex >= uint16(len(feed.Routes)) {
			fmt.Printf("[Warning] Route index %d out of bounds (max %d)\n", schedule.RouteIndex, len(feed.Routes)-1)
			continue
		}
		route := feed.Routes[schedule.RouteIndex]
		headsign := feed.StringTable.Lookup(schedule.HeadsignToken)

		bufService = append(bufService[:0], "service_"...)
		bufService = strconv.AppendUint(bufService, uint64(schedule.ServiceIndex), 10)
		serviceID := string(bufService)

		tripsWriter.Write([]string{
			tripID,
			route.RouteID,
			serviceID,
			headsign,
		})

		// Validate pattern ID bounds
		if schedule.PatternID >= uint32(len(feed.Patterns)) {
			fmt.Printf("[Warning] Pattern ID %d out of bounds (max %d)\n", schedule.PatternID, len(feed.Patterns)-1)
			continue
		}
		pattern := feed.Patterns[schedule.PatternID]
		absoluteTimes := patternAbsoluteTimes[schedule.PatternID]

		for i := 0; i < int(pattern.StopCount); i++ {
			stopIndex := pattern.StopIDTokens[i]

			// Validate stop index bounds
			if stopIndex >= uint16(len(feed.Stops)) {
				fmt.Printf("[Warning] Stop index %d out of bounds (max %d)\n", stopIndex, len(feed.Stops)-1)
				continue
			}
			stop := feed.Stops[stopIndex]

			var arrivalTime, departTime uint32

			if i == 0 {
				departTime = schedule.FirstStopDepartTime
				arrivalTime = departTime
			} else {
				arrivalTime = absoluteTimes[i]
				departTime = absoluteTimes[i]
			}

			arrivalTimeStr := fastTimeToString(bufTimeA, arrivalTime)
			departTimeStr := fastTimeToString(bufTimeD, departTime)

			bufSeq = strconv.AppendUint(bufSeq[:0], uint64(i+1), 10)
			sequenceStr := string(bufSeq)

			stopTimesWriter.Write([]string{
				tripID,
				arrivalTimeStr,
				departTimeStr,
				stop.StopID,
				sequenceStr,
			})
		}
	}

	return nil
}
