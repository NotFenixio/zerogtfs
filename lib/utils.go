package zerogtfs

import "fmt"

func TimeToSeconds(hhmmss string) uint32 {
	var hh, mm, ss int
	_, err := fmt.Sscanf(hhmmss, "%d:%d:%d", &hh, &mm, &ss)
	if err != nil {
		return 0
	}
	return uint32(hh*3600 + mm*60 + ss)
}

func SecondsToTime(seconds uint32) string {
	h := int(seconds / 3600)
	m := int((seconds % 3600) / 60)
	s := int(seconds % 60)
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func EstimateCSVSize(mockData *MockGTFSData) int {
	size := 0
	size += 200 + len(mockData.Agencies)*150
	size += 150 + len(mockData.Stops)*100
	size += 200 + len(mockData.Routes)*120
	size += 150 + len(mockData.Trips)*80
	size += 100 + len(mockData.StopTimes)*50
	return size
}
