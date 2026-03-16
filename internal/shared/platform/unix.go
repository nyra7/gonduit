//go:build !windows

package platform

import (
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const DetachKey = 0x1d
const DetachKeyStr = "Ctrl+]"

func GetOSInfo() (OSInfo, error) {

	info := OSInfo{
		Family: runtime.GOOS,
		Arch:   runtime.GOARCH,
	}

	switch runtime.GOOS {
	case "darwin":
		return getMacOS(info)
	case "linux":
		return getLinux(info)
	default:
		return getUnix(info)
	}
}

func getMacOS(info OSInfo) (OSInfo, error) {
	info.Family = "darwin"

	// sw_vers gives ProductName, ProductVersion, BuildVersion
	out, err := exec.Command("sw_vers").Output()
	if err != nil {
		info.Name = "macOS"
		return info, err
	}

	lines := parseKeyValue(string(out), ":")
	info.Name = lines["ProductName"]
	info.Version = lines["ProductVersion"]
	info.Build = lines["BuildVersion"]

	if info.Name == "" {
		info.Name = "macOS"
	}
	return info, nil
}

func getLinux(info OSInfo) (OSInfo, error) {
	info.Family = "linux"

	// Try /etc/os-release first
	out, err := exec.Command("cat", "/etc/os-release").Output()
	if err == nil {
		lines := parseKeyValue(string(out), "=")
		name := unquote(lines["NAME"])
		version := unquote(lines["VERSION_ID"])
		prettyName := unquote(lines["PRETTY_NAME"])

		if name != "" {
			info.Name = name
		} else if prettyName != "" {
			info.Name = prettyName
		} else {
			info.Name = "Linux"
		}
		info.Version = version
	} else {
		// otherwise fallback on lsb_release
		info.Name, info.Version = lsbRelease()
	}

	// Kernel build string as "build"
	if k, err2 := exec.Command("uname", "-r").Output(); err2 == nil {
		info.Build = strings.TrimSpace(string(k))
	}

	return info, nil
}

func lsbRelease() (name, version string) {
	out, err := exec.Command("lsb_release", "-si").Output()
	if err == nil {
		name = strings.TrimSpace(string(out))
	}
	out, err = exec.Command("lsb_release", "-sr").Output()
	if err == nil {
		version = strings.TrimSpace(string(out))
	}
	if name == "" {
		name = "Linux"
	}
	return
}

func getUnix(info OSInfo) (OSInfo, error) {
	if out, err := exec.Command("uname", "-s").Output(); err == nil {
		info.Name = strings.TrimSpace(string(out))
	} else {
		info.Name = cases.Title(language.English).String(runtime.GOOS)
	}

	if out, err := exec.Command("uname", "-r").Output(); err == nil {
		info.Version = strings.TrimSpace(string(out))
	}

	if out, err := exec.Command("uname", "-v").Output(); err == nil {
		info.Build = strings.TrimSpace(string(out))
	}

	return info, nil
}

// parseKeyValue splits lines of "key<sep>value" into a map
func parseKeyValue(text, sep string) map[string]string {
	m := make(map[string]string)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, sep)
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+len(sep):])
		if key != "" {
			m[key] = val
		}
	}
	return m
}

// unquote removes surrounding double-quotes added by os-release values
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
