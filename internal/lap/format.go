package lap

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func secondsToLapTime(seconds float64) string {
	if seconds < 0 {
		return "-" + secondsToLapTime(-seconds)
	}
	mins := int(seconds) / 60
	secs := seconds - float64(mins*60)
	return fmt.Sprintf("%d:%06.3f", mins, secs)
}

func CarNameForID(carID int, csvPath string) string {
	f, err := os.Open(csvPath)
	if err != nil {
		return fmt.Sprintf("CAR-ID-%d", carID)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = ','
	r.Comment = '#'
	r.FieldsPerRecord = -1

	target := strconv.Itoa(carID)
	for {
		record, err := r.Read()
		if err != nil {
			break
		}
		if len(record) >= 2 && strings.TrimSpace(record[0]) == target {
			return strings.TrimSpace(record[1])
		}
	}
	return fmt.Sprintf("CAR-ID-%d", carID)
}
