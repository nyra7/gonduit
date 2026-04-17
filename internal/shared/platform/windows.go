//go:build windows

package platform

import (
	"fmt"
	"runtime"
	"server/log"
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const DetachKey = 0x1f
const DetachKeyStr = "Ctrl+_"

func GetOSInfo() (OSInfo, error) {
	const keyPath = `SOFTWARE\Microsoft\Windows NT\CurrentVersion`

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return OSInfo{Family: runtime.GOOS, Arch: runtime.GOARCH}, fmt.Errorf("open registry: %w", err)
	}
	defer k.Close()

	// Build number needed early, used for both version string and OS name detection
	major, _, _ := k.GetIntegerValue("CurrentMajorVersionNumber")
	minor, _, _ := k.GetIntegerValue("CurrentMinorVersionNumber")
	currentBuild, _, _ := k.GetStringValue("CurrentBuild")
	ubr, _, _ := k.GetIntegerValue("UBR")

	name, _, err := k.GetStringValue("ProductName")
	if err != nil {
		name = ""
	}

	// Product name (e.g. Windows 10 Pro)
	if buildNum, convErr := strconv.ParseUint(currentBuild, 10, 64); convErr == nil {
		// Replace "10" with "11" in the product name if buildNum > 22000 (since windows shows 10 even if running 11)
		if buildNum >= 22000 {
			name = strings.Replace(name, "Windows 10", "Windows 11", 1)
		}
	}

	// Build version (e.g. 10.0.26100.7840)
	var build string
	build, _, err = k.GetStringValue("LCUVer")
	if err != nil || build == "" {
		if major == 0 && minor == 0 && currentBuild == "" {
			log.Errorf("unable to determine Windows build version. registry keys missing")
			build = "unknown"
		} else {
			build = fmt.Sprintf("%d.%d.%s.%d", major, minor, currentBuild, ubr)
		}
	}

	// Display version (e.g. 23H2)
	version, _, err := k.GetStringValue("DisplayVersion")
	if err != nil {
		log.Errorf("Could not get version: %v", err)
	}

	return OSInfo{
		Family:  runtime.GOOS,
		Name:    name,
		Version: version,
		Build:   build,
		Arch:    runtime.GOARCH,
	}, nil
}
