package ipinfo

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net"
	"net/http"
	"os"
)

type IPInfo struct {
	Ip       string `json:"ip"`
	Hostname string `json:"hostname"`
	Anycast  bool   `json:"anycast"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
}

func InsertAndSelectIPInfoIntoDB(db *sql.DB, ip net.IP) (IPInfo, error) {

	var ipInfo IPInfo
	ipStr := ip.String()
	zap.S().Debugf("Getting IP info for %s", ipStr)
	err := db.QueryRow(
		"SELECT ip, hostname, anycast, city, region, country, loc, org, postal, tz FROM ip_info WHERE ip = $1",
		ipStr).Scan(
		&ipInfo.Ip,
		&ipInfo.Hostname,
		&ipInfo.Anycast,
		&ipInfo.City,
		&ipInfo.Region,
		&ipInfo.Country,
		&ipInfo.Loc,
		&ipInfo.Org,
		&ipInfo.Postal,
		&ipInfo.Timezone)
	// Check if not found
	if errors.Is(err, sql.ErrNoRows) {
		zap.S().Debugf("Cache miss for %s", ipStr)
		ipInfo, err = GetIPInfoFromAPI(ip)
		if err != nil {
			return IPInfo{}, err
		}
		// Insert into database
		_, err = db.Exec(
			"INSERT INTO ip_info (ip, hostname, anycast, city, region, country, loc, org, postal, tz) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) ON CONFLICT DO NOTHING",
			ipInfo.Ip,
			ipInfo.Hostname,
			ipInfo.Anycast,
			ipInfo.City,
			ipInfo.Region,
			ipInfo.Country,
			ipInfo.Loc,
			ipInfo.Org,
			ipInfo.Postal,
			ipInfo.Timezone)
		if err != nil {
			return IPInfo{}, fmt.Errorf("error inserting IP info into database: %s", err)
		}
		return ipInfo, nil
	} else if err != nil {
		return IPInfo{}, fmt.Errorf("error getting IP info from database: %s", err)
	}
	zap.S().Debugf("Cache hit for %s", ipStr)
	return ipInfo, nil
}

func GetIPInfoFromAPI(ip net.IP) (IPInfo, error) {
	ipinfoToken, b := os.LookupEnv("IPINFO_TOKEN")
	if !b {
		zap.S().Fatal("IPINFO_TOKEN not set")
	}

	ipinfoUrl := fmt.Sprintf("https://ipinfo.io/%s/json?token=%s", ip.String(), ipinfoToken)
	resp, err := http.Get(ipinfoUrl)
	if err != nil {
		return IPInfo{}, fmt.Errorf("error getting IP info from API: %s", err)
	}
	defer resp.Body.Close()

	var ipInfo IPInfo
	err = json.NewDecoder(resp.Body).Decode(&ipInfo)
	if err != nil {
		return IPInfo{}, fmt.Errorf("error decoding JSON: %s", err)
	}
	return ipInfo, nil
}
