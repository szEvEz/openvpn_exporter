package openvpn

import (
	"bufio"
	"bytes"
	"io"
	// "net"
	"os"
	"strconv"
	"strings"
	"time"
)

// GlobalStats stores global openvpn statistic information
type GlobalStats struct {
	MaxBcastMcastQueueLen int
}

// Client struct store information from openvpn client statistics
type Client struct {
	RealAddress    string
	VirtualAddress string
	BytesReceived  float64
	BytesSent      float64
	ConnectedSince time.Time
	Username       string
}

// ServerInfo reflects information that was collected about the server
type ServerInfo struct {
	Version        string
	Arch           string
	AdditionalInfo string
}

// Status reflects all information in a status log
type Status struct {
	ClientList  []Client
	GlobalStats GlobalStats
	ServerInfo  ServerInfo
	UpdatedAt   time.Time
}

type parseError struct {
	s string
}

func (e *parseError) Error() string {
	return e.s
}

const (
	timefmt = "Mon Jan 2 15:04:05 2006"
)

// ParseFile parses a openvpn status log and returns respective stats
func ParseFile(statusfile string) (*Status, error) {
	conn, err := os.Open(statusfile)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	status, err := parse(bufio.NewReader(conn))
	if err != nil {
		return nil, err
	}
	return status, nil
}

// func parseIP(ip string) string {
// 	return net.ParseIP(strings.Split(ip, ":")[0]).String()
// }

func parse(reader *bufio.Reader) (*Status, error) {
	buf, _ := reader.Peek(19)

	if bytes.HasPrefix(buf, []byte("TITLE,OpenVPN")) {
		return parseStatusV2AndV3(reader, ",")
	}
	if bytes.HasPrefix(buf, []byte("TITLE\tOpenVPN")) {
		return parseStatusV2AndV3(reader, "\t")
	}
	return nil, &parseError{"bad status file"}
}

func parseStatusV2AndV3(reader io.Reader, separator string) (*Status, error) {
	scanner := bufio.NewScanner(reader)
	var maxBcastMcastQueueLen int
	var lastUpdatedAt time.Time
	var clients []Client
	var serverInfo ServerInfo
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), separator)
		if fields[0] == "TIME" && len(fields) == 3 {
			updatedAtInt, _ := strconv.ParseInt(fields[2], 10, 64)
			lastUpdatedAt = time.Unix(updatedAtInt, 0)
		} else if fields[0] == "CLIENT_LIST" {
			bytesRec, _ := strconv.ParseFloat(fields[5], 64)
			bytesSent, _ := strconv.ParseFloat(fields[6], 64)
			connectedSinceInt, _ := strconv.ParseInt(fields[8], 10, 64)
			client := Client{
				RealAddress:    fields[2],
				VirtualAddress: fields[3],
				BytesReceived:  bytesRec,
				BytesSent:      bytesSent,
				ConnectedSince: time.Unix(connectedSinceInt, 0),
				Username:       fields[9],
			}
			clients = append(clients, client)
		} else if fields[0] == "GLOBAL_STATS" {
			i, err := strconv.Atoi(fields[2])
			if err == nil {
				maxBcastMcastQueueLen = i
			}
		} else if fields[0] == "TITLE" {
			infoFields := strings.Split(fields[1], " ")
			serverInfo = ServerInfo{
				Version:        infoFields[1],
				Arch:           infoFields[2],
				AdditionalInfo: strings.Join(infoFields[3:], " "),
			}
		}
	}
	return &Status{
		GlobalStats: GlobalStats{maxBcastMcastQueueLen},
		UpdatedAt:   lastUpdatedAt,
		ClientList:  clients,
		ServerInfo:  serverInfo,
	}, nil
}
