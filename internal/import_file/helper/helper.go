package helper

import (
	"fmt"
	"strings"
)

func SafeGet(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func CleanNumber(val string) string {
	val = strings.ReplaceAll(val, ",", "")
	return val
}

func ParsePeriod(input string) string {
	parts := strings.Split(input, "-")
	if len(parts) != 2 {
		return ""
	}

	monthMap := map[string]string{
		"Jan": "01", "Feb": "02", "Mar": "03",
		"Apr": "04", "May": "05", "Jun": "06",
		"Jul": "07", "Aug": "08", "Sep": "09",
		"Oct": "10", "Nov": "11", "Dec": "12",
	}

	month, ok := monthMap[parts[0]]
	if !ok {
		return ""
	}

	year := "20" + parts[1]

	return fmt.Sprintf("%s-%s", year, month)
}
