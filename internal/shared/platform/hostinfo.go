package platform

import (
	"net"
	"os"
	"os/user"
	"server/log"
	"shared/proto"
	"shared/util"
	"strconv"
)

type HostInfo struct {
	Hostname       string
	Username       string
	WorkingDir     string
	Ip             string
	Port           string
	NetworkAdapter string
	OsInfo         OSInfo
}

func GetHostInfo(addr net.Addr) HostInfo {

	osInfo, osErr := GetOSInfo()

	if osErr != nil {
		log.Errorf("error while fetching OS info: %s", osErr.Error())
	}

	// Defaults
	info := HostInfo{
		Hostname:       "?",
		Username:       "?",
		WorkingDir:     "?",
		NetworkAdapter: "?",
		OsInfo:         osInfo,
	}

	// Local IP + interface
	if ip, itf, err := util.LocalIPAndInterface(addr); err == nil {
		info.Ip = ip.String()
		info.NetworkAdapter = itf.Name
	}

	// Hostname
	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	} else if info.Ip != "" {
		info.Hostname = info.Ip
	}

	// Username
	if u, err := user.Current(); err == nil {
		info.Username = u.Username
	}

	// Working directory
	if wd, err := os.Getwd(); err == nil {
		info.WorkingDir = wd
	}

	// Port
	if tcp, ok := addr.(*net.TCPAddr); ok {
		info.Port = strconv.Itoa(tcp.Port)
	}

	return info

}

func (h HostInfo) ToProto() *proto.HostInfo {
	return &proto.HostInfo{
		Hostname:       h.Hostname,
		Username:       h.Username,
		WorkingDir:     h.WorkingDir,
		Ip:             h.Ip,
		Port:           h.Port,
		NetworkAdapter: h.NetworkAdapter,
		OsInfo:         h.OsInfo.ToProto(),
	}
}

func HostInfoFromProto(info *proto.HostInfo) HostInfo {
	return HostInfo{
		Hostname:       info.Hostname,
		Username:       info.Username,
		WorkingDir:     info.WorkingDir,
		Ip:             info.Ip,
		Port:           info.Port,
		NetworkAdapter: info.NetworkAdapter,
		OsInfo:         OSInfoFromProto(info.OsInfo),
	}
}
