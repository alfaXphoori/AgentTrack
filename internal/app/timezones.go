package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ListAllTimezones returns a list of all IANA timezones found in the system
func ListAllTimezones() []string {
	zonesMap := make(map[string]bool)

	// Common paths for zoneinfo on various systems
	zoneDirs := []string{
		"/usr/share/zoneinfo",
		"/usr/share/lib/zoneinfo",
		"/usr/lib/zoneinfo",
	}

	found := false
	for _, zoneDir := range zoneDirs {
		if _, err := os.Stat(zoneDir); err == nil {
			filepath.Walk(zoneDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}

				relPath, err := filepath.Rel(zoneDir, path)
				if err != nil {
					return nil
				}

				// Skip technical files and specific aliases
				if strings.HasPrefix(relPath, "posix/") ||
					strings.HasPrefix(relPath, "right/") ||
					strings.Contains(relPath, ".") ||
					relPath == "+VERSION" ||
					relPath == "iso3166.tab" ||
					relPath == "zone.tab" ||
					relPath == "zone1970.tab" {
					return nil
				}

				// We want Africa/, America/, Asia/, Atlantic/, Australia/, Europe/, Indian/, Pacific/, Etc/, UTC
				zonesMap[relPath] = true
				found = true
				return nil
			})
		}
	}

	if !found {
		// Minimum fallback list
		return []string{"UTC", "Asia/Bangkok", "Asia/Tokyo", "Asia/Singapore", "Asia/Hong_Kong", "Asia/Seoul",
			"America/New_York", "America/Los_Angeles", "America/Chicago", "Europe/London", "Europe/Paris", "Europe/Berlin",
			"Australia/Sydney", "Pacific/Auckland"}
	}

	zones := make([]string, 0, len(zonesMap))
	for z := range zonesMap {
		zones = append(zones, z)
	}

	sort.Strings(zones)
	return zones
}
