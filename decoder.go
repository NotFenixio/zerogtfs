package main

func DecodeGTFSFromFile(filename string) (*DecodedGTFSFeed, error) {

	mockData := NewMockData()

	decoded := &DecodedGTFSFeed{
		Feed:        mockData,
		StopIndex:   make(map[string]*MockStop),
		RouteIndex:  make(map[string]*MockRoute),
		TripIndex:   make(map[string]*MockTrip),
		AgencyIndex: make(map[string]*MockAgency),
	}

	for _, stop := range mockData.Stops {
		decoded.StopIndex[stop.StopId] = stop
	}

	for _, route := range mockData.Routes {
		decoded.RouteIndex[route.RouteId] = route
	}

	for _, trip := range mockData.Trips {
		decoded.TripIndex[trip.TripId] = trip
	}

	for _, agency := range mockData.Agencies {
		decoded.AgencyIndex[agency.AgencyId] = agency
	}

	return decoded, nil
}
