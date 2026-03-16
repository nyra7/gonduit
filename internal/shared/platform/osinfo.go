package platform

import (
	"fmt"
	"shared/proto"
)

// OSInfo holds the detected OS details
type OSInfo struct {
	// Family represents the OS family (e.g. darwin, windows, linux, etc.)
	Family string

	// Name represents the OS / Distribution name (e.g. macOS, Ubuntu, etc.)
	Name string

	// Version represents the OS / Distribution version (e.g. 24.04 for Ubuntu, 26.3 for macOS, etc.)
	Version string

	// Build represents the OS build number / kernel version (e.g. 23E224 for macOS, 22631 for Windows, etc.)
	Build string

	// Arch represents the architecture of the running program (e.g. arm64, amd64, etc.)
	Arch string
}

func (o OSInfo) String() string {
	s := fmt.Sprintf("%s %s", o.Name, o.Version)
	if o.Build != "" {
		s += fmt.Sprintf(" (build %s)", o.Build)
	}
	s += fmt.Sprintf(" [%s/%s]", o.Family, o.Arch)
	return s
}

func (o OSInfo) ToProto() *proto.OSInfo {
	return &proto.OSInfo{
		Family:  o.Family,
		Name:    o.Name,
		Version: o.Version,
		Build:   o.Build,
		Arch:    o.Arch,
	}
}

func OSInfoFromProto(info *proto.OSInfo) OSInfo {
	return OSInfo{
		Family:  info.Family,
		Name:    info.Name,
		Version: info.Version,
		Build:   info.Build,
		Arch:    info.Arch,
	}
}
