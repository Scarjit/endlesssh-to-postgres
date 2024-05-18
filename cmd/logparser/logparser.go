package logparser

import (
	"bufio"
	"database/sql"
	"fmt"
	"go.uber.org/zap"
	"net"
	"os"
	"parseEndlessSSH/cmd/ipinfo"
	"regexp"
	"sort"
	"strconv"
	"time"
)

func ParseLog(db *sql.DB) error {
	file, err := os.Open("/endlessh.log")
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	var lines []LogLine
	for scanner.Scan() {
		var ll LogLine
		ll, err = parseLine(scanner.Text())
		if err != nil {
			continue
		}
		lines = append(lines, ll)
	}

	// remove duplicates by line.Host
	sort.SliceStable(
		lines, func(i, j int) bool {
			return lines[i].Host.String() < lines[j].Host.String()
		})

	lines = removeDuplicateLL(lines)

	for _, line := range lines {
		HandleLine(line, db)
	}

	// Reset log file
	file.Close()
	err = os.Truncate("/endlessh.log", 0)
	if err != nil {
		return err
	}

	return nil
}

func HandleLine(line LogLine, db *sql.DB) {
	_, err := ipinfo.InsertAndSelectIPInfoIntoDB(db, line.Host)
	if err != nil {
		zap.S().Errorf("Error getting IP info: %s", err)
	}

	_, err = db.Exec(
		"INSERT INTO connections (date, ip_info_id, duration, bytes) VALUES ($1, (SELECT id FROM ip_info WHERE ip = $2), $3, $4)",
		line.Date, line.Host.String(), line.Duration.Seconds(), line.Bytes)
	if err != nil {
		zap.S().Errorf("Error inserting into database: %s", err)
	}

}

func removeDuplicateLL(strSlice []LogLine) []LogLine {
	allKeys := make(map[string]bool)
	var list []LogLine
	for _, item := range strSlice {
		if _, value := allKeys[item.Host.String()]; !value {
			allKeys[item.Host.String()] = true
			list = append(list, item)
		}
	}
	return list
}

var lineRegex = regexp.MustCompile(`(?P<date>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z) CLOSE host=(?P<host>[:.a-f0-9]+) port=\d+ fd=\d+ time=(?P<time>[\d.]+) bytes=(?P<bytes>\d+)`)

type LogLine struct {
	Date     time.Time
	Host     net.IP
	Duration time.Duration
	Bytes    uint64
}

func parseLine(text string) (LogLine, error) {
	submatch := lineRegex.FindStringSubmatch(text)
	if submatch == nil {
		return LogLine{}, fmt.Errorf("invalid line: %s", text)
	}
	result := make(map[string]string)
	for i, name := range lineRegex.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = submatch[i]
		}
	}
	var err error
	var ll LogLine
	ll.Date, err = time.Parse(time.RFC3339, result["date"])
	if err != nil {
		return LogLine{}, err
	}
	ll.Host = net.ParseIP(result["host"])
	ll.Duration, err = time.ParseDuration(fmt.Sprintf("%ss", result["time"]))
	if err != nil {
		return LogLine{}, err
	}
	ll.Bytes, err = strconv.ParseUint(result["bytes"], 10, 64)
	if err != nil {
		return LogLine{}, err
	}

	return ll, nil
}
