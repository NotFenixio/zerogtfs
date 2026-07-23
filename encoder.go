package main

func NewMockData() *MockGTFSData {

	return createMockData()
}

func createMockData() *MockGTFSData {
	data := &MockGTFSData{}

	data.Agencies = []*MockAgency{
		{
			AgencyId: "AGENCY_1",
			Name:     "Metro Transit",
			Url:      "https://metro.example.com",
			Timezone: "America/Chicago",
			Lang:     "en",
			Phone:    "(555) 555-0100",
		},
		{
			AgencyId: "AGENCY_2",
			Name:     "Community Bus",
			Url:      "https://community.example.com",
			Timezone: "America/Denver",
			Lang:     "en",
		},
	}

	stopNames := []string{
		"Downtown Station",
		"Central Park",
		"University Avenue",
		"Hospital Complex",
		"Shopping Center",
		"Residential Area",
		"Industrial Park",
		"Airport Terminal",
		"Transit Center",
		"Final Destination",
	}

	for i, name := range stopNames {
		data.Stops = append(data.Stops, &MockStop{
			StopId:       "STOP_" + string(rune('0'+i+1)),
			Name:         name,
			Lat:          40.7128 + float64(i)*0.01,
			Lon:          -74.0060 + float64(i)*0.01,
			Code:         "S" + string(rune('0'+i+1)),
			LocationType: 0,
		})
	}

	data.Routes = []*MockRoute{
		{
			RouteId:   "ROUTE_1",
			AgencyId:  "AGENCY_1",
			ShortName: "1",
			LongName:  "Downtown - Airport Express",
			Type:      3,
			Url:       "https://metro.example.com/routes/1",
			Color:     0xFF0000,
			TextColor: 0xFFFFFF,
		},
		{
			RouteId:   "ROUTE_2",
			AgencyId:  "AGENCY_2",
			ShortName: "42",
			LongName:  "Industrial District Local",
			Type:      3,
			Url:       "https://community.example.com/routes/42",
			Color:     0x0000FF,
			TextColor: 0xFFFFFF,
		},
	}

	data.Trips = []*MockTrip{
		{
			TripId:               "TRIP_001",
			RouteId:              "ROUTE_1",
			ServiceId:            "WEEKDAY",
			ShapeId:              "SHAPE_1",
			DirectionId:          0,
			Headsign:             "Airport",
			WheelchairAccessible: "1",
		},
		{
			TripId:               "TRIP_002",
			RouteId:              "ROUTE_1",
			ServiceId:            "WEEKEND",
			ShapeId:              "SHAPE_1",
			DirectionId:          1,
			Headsign:             "Downtown",
			WheelchairAccessible: "0",
		},
		{
			TripId:               "TRIP_003",
			RouteId:              "ROUTE_2",
			ServiceId:            "WEEKDAY",
			ShapeId:              "SHAPE_2",
			DirectionId:          0,
			Headsign:             "Industrial Park",
			WheelchairAccessible: "1",
		},
	}

	baseDeparture := uint32(8 * 3600)
	for i := 0; i < 10; i++ {
		arrivalTime := baseDeparture + uint32(i*5*60)
		departureTime := arrivalTime + uint32(2*60)
		data.StopTimes = append(data.StopTimes, &MockStopTime{
			TripId:       "TRIP_001",
			StopId:       "STOP_" + string(rune('0'+i+1)),
			StopSequence: uint32(i + 1),
			ArrivalTime:  arrivalTime,
			DepartTime:   departureTime,
			PickupType:   0,
			DropOffType:  0,
		})
	}

	baseDeparture = uint32(14 * 3600)
	for i := 0; i < 10; i++ {
		arrivalTime := baseDeparture + uint32(i*5*60)
		departureTime := arrivalTime + uint32(2*60)
		data.StopTimes = append(data.StopTimes, &MockStopTime{
			TripId:       "TRIP_002",
			StopId:       "STOP_" + string(rune('0'+(10-i))),
			StopSequence: uint32(i + 1),
			ArrivalTime:  arrivalTime,
			DepartTime:   departureTime,
			PickupType:   0,
			DropOffType:  0,
		})
	}

	baseDeparture = uint32(10 * 3600)
	for i := 0; i < 10; i++ {
		arrivalTime := baseDeparture + uint32(i*4*60)
		departureTime := arrivalTime + uint32(1*60)
		data.StopTimes = append(data.StopTimes, &MockStopTime{
			TripId:       "TRIP_003",
			StopId:       "STOP_" + string(rune('0'+i+1)),
			StopSequence: uint32(i + 1),
			ArrivalTime:  arrivalTime,
			DepartTime:   departureTime,
			PickupType:   0,
			DropOffType:  0,
		})
	}

	return data
}
