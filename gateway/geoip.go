package gateway

import (
	"encoding/csv"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

type GeoIPEntry struct {
	StartIP     net.IP
	EndIP       net.IP
	Country     string
	CountryCode string
}

type GeoIPService struct {
	entries []GeoIPEntry
	mu      sync.RWMutex
}

func NewGeoIPService() *GeoIPService {
	return &GeoIPService{}
}

// LoadCSV loads a GeoIP database from a CSV file.
// Expected format: start_ip,end_ip,country,country_code
func (g *GeoIPService) LoadCSV(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open GeoIP database: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to parse GeoIP CSV: %w", err)
	}

	var entries []GeoIPEntry
	for i, row := range records {
		if i == 0 && strings.ToLower(row[0]) == "start_ip" {
			continue
		}
		if len(row) < 3 {
			continue
		}

		startIP := net.ParseIP(row[0])
		endIP := net.ParseIP(row[1])
		if startIP == nil || endIP == nil {
			continue
		}

		countryCode := ""
		if len(row) >= 4 {
			countryCode = row[3]
		}

		entries = append(entries, GeoIPEntry{
			StartIP:     startIP,
			EndIP:       endIP,
			Country:     row[2],
			CountryCode: countryCode,
		})
	}

	g.mu.Lock()
	g.entries = entries
	g.mu.Unlock()

	return nil
}

// Lookup returns the country code for a given IP address.
func (g *GeoIPService) Lookup(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return ""
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, entry := range entries4 {
		if ipInRange(ip4, entry.StartIP, entry.EndIP) {
			return entry.CountryCode
		}
	}

	return ""
}

func ipInRange(ip, start, end net.IP) bool {
	ip4 := ip.To4()
	start4 := start.To4()
	end4 := end.To4()
	if ip4 == nil || start4 == nil || end4 == nil {
		return false
	}

	ipInt := ip4toInt(ip4)
	startInt := ip4toInt(start4)
	endInt := ip4toInt(end4)

	return ipInt >= startInt && ipInt <= endInt
}

func ip4toInt(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

// Default well-known IP ranges for common countries.
var entries4 = []GeoIPEntry{
	{net.ParseIP("1.0.0.0"), net.ParseIP("1.255.255.255"), "Australia", "AU"},
	{net.ParseIP("3.0.0.0"), net.ParseIP("3.255.255.255"), "United States", "US"},
	{net.ParseIP("8.0.0.0"), net.ParseIP("8.255.255.255"), "United States", "US"},
	{net.ParseIP("13.0.0.0"), net.ParseIP("13.255.255.255"), "United States", "US"},
	{net.ParseIP("18.0.0.0"), net.ParseIP("18.255.255.255"), "United States", "US"},
	{net.ParseIP("23.0.0.0"), net.ParseIP("23.255.255.255"), "United States", "US"},
	{net.ParseIP("24.0.0.0"), net.ParseIP("24.255.255.255"), "United States", "US"},
	{net.ParseIP("34.0.0.0"), net.ParseIP("35.255.255.255"), "United States", "US"},
	{net.ParseIP("44.0.0.0"), net.ParseIP("44.255.255.255"), "United States", "US"},
	{net.ParseIP("50.0.0.0"), net.ParseIP("50.255.255.255"), "United States", "US"},
	{net.ParseIP("51.0.0.0"), net.ParseIP("51.255.255.255"), "United Kingdom", "GB"},
	{net.ParseIP("52.0.0.0"), net.ParseIP("52.255.255.255"), "United States", "US"},
	{net.ParseIP("54.0.0.0"), net.ParseIP("54.255.255.255"), "United States", "US"},
	{net.ParseIP("58.0.0.0"), net.ParseIP("61.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("62.0.0.0"), net.ParseIP("62.255.255.255"), "Europe", "EU"},
	{net.ParseIP("77.0.0.0"), net.ParseIP("95.255.255.255"), "Europe", "EU"},
	{net.ParseIP("96.0.0.0"), net.ParseIP("96.255.255.255"), "United States", "US"},
	{net.ParseIP("98.0.0.0"), net.ParseIP("98.255.255.255"), "United States", "US"},
	{net.ParseIP("100.0.0.0"), net.ParseIP("100.255.255.255"), "United States", "US"},
	{net.ParseIP("101.0.0.0"), net.ParseIP("103.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("104.0.0.0"), net.ParseIP("104.255.255.255"), "United States", "US"},
	{net.ParseIP("106.0.0.0"), net.ParseIP("106.255.255.255"), "China", "CN"},
	{net.ParseIP("108.0.0.0"), net.ParseIP("108.255.255.255"), "United States", "US"},
	{net.ParseIP("109.0.0.0"), net.ParseIP("109.255.255.255"), "Europe", "EU"},
	{net.ParseIP("110.0.0.0"), net.ParseIP("126.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("128.0.0.0"), net.ParseIP("132.255.255.255"), "United States", "US"},
	{net.ParseIP("134.0.0.0"), net.ParseIP("139.255.255.255"), "United States", "US"},
	{net.ParseIP("140.0.0.0"), net.ParseIP("140.255.255.255"), "United States", "US"},
	{net.ParseIP("141.0.0.0"), net.ParseIP("141.255.255.255"), "Europe", "EU"},
	{net.ParseIP("142.0.0.0"), net.ParseIP("142.255.255.255"), "United States", "US"},
	{net.ParseIP("143.0.0.0"), net.ParseIP("143.255.255.255"), "United States", "US"},
	{net.ParseIP("144.0.0.0"), net.ParseIP("144.255.255.255"), "United States", "US"},
	{net.ParseIP("145.0.0.0"), net.ParseIP("145.255.255.255"), "Europe", "EU"},
	{net.ParseIP("146.0.0.0"), net.ParseIP("146.255.255.255"), "United States", "US"},
	{net.ParseIP("147.0.0.0"), net.ParseIP("147.255.255.255"), "United States", "US"},
	{net.ParseIP("148.0.0.0"), net.ParseIP("148.255.255.255"), "United States", "US"},
	{net.ParseIP("149.0.0.0"), net.ParseIP("149.255.255.255"), "United States", "US"},
	{net.ParseIP("150.0.0.0"), net.ParseIP("150.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("151.0.0.0"), net.ParseIP("151.255.255.255"), "Europe", "EU"},
	{net.ParseIP("152.0.0.0"), net.ParseIP("152.255.255.255"), "United States", "US"},
	{net.ParseIP("153.0.0.0"), net.ParseIP("153.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("154.0.0.0"), net.ParseIP("154.255.255.255"), "Africa", "AF"},
	{net.ParseIP("155.0.0.0"), net.ParseIP("155.255.255.255"), "United States", "US"},
	{net.ParseIP("156.0.0.0"), net.ParseIP("156.255.255.255"), "United States", "US"},
	{net.ParseIP("157.0.0.0"), net.ParseIP("157.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("158.0.0.0"), net.ParseIP("158.255.255.255"), "United States", "US"},
	{net.ParseIP("159.0.0.0"), net.ParseIP("159.255.255.255"), "United States", "US"},
	{net.ParseIP("160.0.0.0"), net.ParseIP("164.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("165.0.0.0"), net.ParseIP("166.255.255.255"), "United States", "US"},
	{net.ParseIP("167.0.0.0"), net.ParseIP("167.255.255.255"), "United States", "US"},
	{net.ParseIP("168.0.0.0"), net.ParseIP("168.255.255.255"), "United States", "US"},
	{net.ParseIP("169.0.0.0"), net.ParseIP("169.255.255.255"), "United States", "US"},
	{net.ParseIP("170.0.0.0"), net.ParseIP("170.255.255.255"), "United States", "US"},
	{net.ParseIP("171.0.0.0"), net.ParseIP("171.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("172.0.0.0"), net.ParseIP("172.255.255.255"), "United States", "US"},
	{net.ParseIP("173.0.0.0"), net.ParseIP("174.255.255.255"), "United States", "US"},
	{net.ParseIP("175.0.0.0"), net.ParseIP("175.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("176.0.0.0"), net.ParseIP("176.255.255.255"), "Europe", "EU"},
	{net.ParseIP("177.0.0.0"), net.ParseIP("177.255.255.255"), "Latin America", "LA"},
	{net.ParseIP("178.0.0.0"), net.ParseIP("178.255.255.255"), "Europe", "EU"},
	{net.ParseIP("179.0.0.0"), net.ParseIP("179.255.255.255"), "Latin America", "LA"},
	{net.ParseIP("180.0.0.0"), net.ParseIP("180.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("181.0.0.0"), net.ParseIP("181.255.255.255"), "Latin America", "LA"},
	{net.ParseIP("182.0.0.0"), net.ParseIP("182.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("183.0.0.0"), net.ParseIP("183.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("184.0.0.0"), net.ParseIP("184.255.255.255"), "United States", "US"},
	{net.ParseIP("185.0.0.0"), net.ParseIP("185.255.255.255"), "Europe", "EU"},
	{net.ParseIP("186.0.0.0"), net.ParseIP("186.255.255.255"), "Latin America", "LA"},
	{net.ParseIP("187.0.0.0"), net.ParseIP("187.255.255.255"), "Latin America", "LA"},
	{net.ParseIP("188.0.0.0"), net.ParseIP("188.255.255.255"), "Europe", "EU"},
	{net.ParseIP("189.0.0.0"), net.ParseIP("189.255.255.255"), "Latin America", "LA"},
	{net.ParseIP("190.0.0.0"), net.ParseIP("190.255.255.255"), "Latin America", "LA"},
	{net.ParseIP("191.0.0.0"), net.ParseIP("191.255.255.255"), "Latin America", "LA"},
	{net.ParseIP("192.0.0.0"), net.ParseIP("192.255.255.255"), "United States", "US"},
	{net.ParseIP("193.0.0.0"), net.ParseIP("195.255.255.255"), "Europe", "EU"},
	{net.ParseIP("196.0.0.0"), net.ParseIP("196.255.255.255"), "Africa", "AF"},
	{net.ParseIP("197.0.0.0"), net.ParseIP("197.255.255.255"), "Africa", "AF"},
	{net.ParseIP("198.0.0.0"), net.ParseIP("198.255.255.255"), "United States", "US"},
	{net.ParseIP("199.0.0.0"), net.ParseIP("199.255.255.255"), "United States", "US"},
	{net.ParseIP("200.0.0.0"), net.ParseIP("201.255.255.255"), "Latin America", "LA"},
	{net.ParseIP("202.0.0.0"), net.ParseIP("203.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("204.0.0.0"), net.ParseIP("209.255.255.255"), "United States", "US"},
	{net.ParseIP("210.0.0.0"), net.ParseIP("210.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("211.0.0.0"), net.ParseIP("211.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("212.0.0.0"), net.ParseIP("212.255.255.255"), "Europe", "EU"},
	{net.ParseIP("213.0.0.0"), net.ParseIP("213.255.255.255"), "Europe", "EU"},
	{net.ParseIP("214.0.0.0"), net.ParseIP("215.255.255.255"), "United States", "US"},
	{net.ParseIP("216.0.0.0"), net.ParseIP("216.255.255.255"), "United States", "US"},
	{net.ParseIP("217.0.0.0"), net.ParseIP("217.255.255.255"), "Europe", "EU"},
	{net.ParseIP("218.0.0.0"), net.ParseIP("218.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("219.0.0.0"), net.ParseIP("219.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("220.0.0.0"), net.ParseIP("220.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("221.0.0.0"), net.ParseIP("221.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("222.0.0.0"), net.ParseIP("223.255.255.255"), "Asia Pacific", "AP"},
	{net.ParseIP("224.0.0.0"), net.ParseIP("255.255.255.255"), "Reserved", "XX"},
}
