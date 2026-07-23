package main

type MockGTFSData struct {
	Agencies  []*MockAgency
	Stops     []*MockStop
	Routes    []*MockRoute
	Trips     []*MockTrip
	StopTimes []*MockStopTime
}

type MockAgency struct {
	AgencyId string
	Name     string
	Url      string
	Timezone string
	Lang     string
	Phone    string
	FareUrl  string
}

type MockStop struct {
	StopId        string
	Name          string
	Lat           float64
	Lon           float64
	Code          string
	Desc          string
	ZoneId        string
	Url           string
	LocationType  int32
	ParentStation string
}

type MockRoute struct {
	RouteId   string
	AgencyId  string
	ShortName string
	LongName  string
	Desc      string
	Type      int32
	Url       string
	Color     uint32
	TextColor uint32
	SortOrder int32
}

type MockTrip struct {
	TripId               string
	RouteId              string
	ServiceId            string
	ShapeId              string
	DirectionId          int32
	Headsign             string
	BlockId              int32
	WheelchairAccessible string
}

type MockStopTime struct {
	TripId       string
	StopId       string
	StopSequence uint32
	ArrivalTime  uint32
	DepartTime   uint32
	PickupType   int32
	DropOffType  int32
}

type DecodedGTFSFeed struct {
	Feed        *MockGTFSData
	StopIndex   map[string]*MockStop
	RouteIndex  map[string]*MockRoute
	TripIndex   map[string]*MockTrip
	AgencyIndex map[string]*MockAgency
}

type TripStopDetail struct {
	Stop          *MockStop
	StopSequence  uint32
	ArrivalTime   string
	DepartureTime string
}
