package scanner

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// ScanJob represents a single port scanning task.
type ScanJob struct {
	Host string
	Port int
}

// ScanResult represents the outcome of a port scan attempt.
type ScanResult struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	State   string `json:"state"`
	Service string `json:"service,omitempty"`
}

// grabBanner retrieves service banner based on port protocol.
// Implements protocol-specific communication strategies:
// - HTTP ports: send GET request and read response
// - SSH/FTP/SMTP: read auto-generated banner
// - Others: return empty string
func grabBanner(conn net.Conn, port int) string {
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	var banner string

	switch port {
	case 80, 443, 8080, 8443, 3000, 5000:
		banner = fetchHTTPBanner(conn)
	case 21, 22, 25, 110, 143:
		banner = readAutoBanner(conn)
	default:
		return ""
	}

	return cleanBanner(banner)
}

// fetchHTTPBanner sends HTTP GET request and reads the response.
func fetchHTTPBanner(conn net.Conn) string {
	_, err := conn.Write([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))
	if err != nil {
		return ""
	}

	buffer := make([]byte, 512)
	n, err := conn.Read(buffer)
	if err != nil {
		return ""
	}

	return string(buffer[:n])
}

// readAutoBanner reads banner sent by service upon connection.
// Used for SSH, FTP, SMTP, POP3, IMAP protocols.
func readAutoBanner(conn net.Conn) string {
	buffer := make([]byte, 512)
	n, err := conn.Read(buffer)
	if err != nil {
		return ""
	}

	return string(buffer[:n])
}

// cleanBanner normalizes banner format by replacing line endings.
func cleanBanner(banner string) string {
	cleaned := strings.ReplaceAll(banner, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")
	return cleaned
}

// Worker processes scan jobs using TCP Connect Scan.
// Establishes full connection and retrieves service banner if available.
func Worker(jobs <-chan ScanJob, results chan<- ScanResult) {
	for job := range jobs {
		address := job.Host + ":" + strconv.Itoa(job.Port)
		conn, err := net.DialTimeout("tcp", address, 2*time.Second)

		var result ScanResult
		if err != nil {
			result = ScanResult{Host: job.Host, Port: job.Port, State: "Closed"}
		} else {
			banner := grabBanner(conn, job.Port)
			result = ScanResult{Host: job.Host, Port: job.Port, State: "Open", Service: banner}
			_ = conn.Close()
		}
		results <- result
	}
}

// SynWorker processes scan jobs using SYN Scan (stealth mode).
// Sends SYN packet and detects response without completing handshake.
// Requires root/administrator privileges on Unix-like systems.
func SynWorker(jobs <-chan ScanJob, results chan<- ScanResult) {
	for job := range jobs {
		state := performSynScan(job.Host, job.Port)
		result := ScanResult{Host: job.Host, Port: job.Port, State: state}
		results <- result
	}
}

// performSynScan attempts SYN scan on a single host:port combination.
// Uses raw socket to send SYN packet and listen for SYN-ACK response.
// Returns "Open" or "Closed" based on TCP response received.
func performSynScan(host string, port int) string {
	// 1. Find all network interfaces.
	ifaces, err := net.Interfaces()
	if err != nil {
		return "Closed"
	}

	var srcIP net.IP
	var device *net.Interface

	// 2. Find suitable source IP address (not loopback, not down).
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					srcIP = ipnet.IP.To4()
					device = &iface
					break
				}
			}
		}
		if srcIP != nil {
			break
		}
	}

	if srcIP == nil || device == nil {
		return "Closed"
	}

	// 3. Resolve target IP address.
	dstIPs, err := net.LookupIP(host)
	if err != nil {
		return "Closed"
	}

	dstIP := dstIPs[0].To4()
	if dstIP == nil {
		return "Closed"
	}

	// 4. Open pcap handle for packet capture/transmission.
	handle, err := pcap.OpenLive(device.Name, 65535, false, 2*time.Second)
	if err != nil {
		return "Closed"
	}
	defer handle.Close()

	// 5. Set BPF filter to capture only relevant responses.
	filter := fmt.Sprintf("tcp and src host %s and src port %d and dst host %s", dstIP.String(), port, srcIP.String())
	if err := handle.SetBPFFilter(filter); err != nil {
		return "Closed"
	}

	// 6. Create TCP SYN packet layers.
	srcPort := uint16(rand.Intn(65535-1024) + 1024)
	dstPort := uint16(port)

	ipLayer := &layers.IPv4{
		SrcIP:    srcIP,
		DstIP:    dstIP,
		Protocol: layers.IPProtocolTCP,
		TTL:      64,
	}

	tcpLayer := &layers.TCP{
		SrcPort: layers.TCPPort(srcPort),
		DstPort: layers.TCPPort(dstPort),
		SYN:     true,
		Seq:     1105024978,
	}

	// 7. Calculate TCP checksum with IP layer context.
	_ = tcpLayer.SetNetworkLayerForChecksum(ipLayer)

	// 8. Serialize packet into buffer.
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(buffer, opts, ipLayer, tcpLayer); err != nil {
		return "Closed"
	}

	// 9. Send SYN packet.
	if err := handle.WritePacketData(buffer.Bytes()); err != nil {
		return "Closed"
	}

	// 10. Listen for response (SYN-ACK or RST).
	timeout := time.After(2 * time.Second)
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	for {
		select {
		case packet := <-packetSource.Packets():
			if packet == nil {
				return "Closed"
			}

			// Check TCP layer for SYN-ACK or RST flags.
			if tcpPacket, ok := packet.Layer(layers.LayerTypeTCP).(*layers.TCP); ok {
				if tcpPacket.SYN && tcpPacket.ACK {
					return "Open"
				}
				if tcpPacket.RST {
					return "Closed"
				}
			}

		case <-timeout:
			return "Closed"
		}
	}
}

// InitSynScan initializes resources needed for SYN scanning.
// Validates that pcap library is available and privileges are sufficient.
// Returns error if SYN scan prerequisites are not met.
func InitSynScan() error {
	// Attempt to list network devices (requires elevated privileges).
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return fmt.Errorf("SYN scan requires root/administrator privileges and libpcap: %v", err)
	}

	if len(devices) == 0 {
		return fmt.Errorf("no network devices found for SYN scan")
	}

	return nil
}

// UdpWorker processes scan jobs using UDP Scan.
// Sends UDP packets and detects responses or ICMP errors.
func UdpWorker(jobs <-chan ScanJob, results chan<- ScanResult) {
	for job := range jobs {
		state := performUdpScan(job.Host, job.Port)
		result := ScanResult{Host: job.Host, Port: job.Port, State: state}
		results <- result
	}
}

// performUdpScan attempts UDP scan on a single host:port combination.
// Sends UDP packet and listens for service response or ICMP errors.
// Returns "Open" (service responded), "Closed" (ICMP unreachable), or "Open|Filtered" (timeout).
func performUdpScan(host string, port int) string {
	address := host + ":" + strconv.Itoa(port)

	// 1. Dial UDP connection with timeout.
	conn, err := net.DialTimeout("udp", address, 2*time.Second)
	if err != nil {
		// Check if it's a timeout error using net.Error interface.
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return "Open|Filtered"
		}
		// Any other error (like connection refused) indicates closed port.
		return "Closed"
	}
	defer conn.Close()

	// 2. Set read deadline for receiving responses.
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// 3. Send probe packet.
	_, err = conn.Write([]byte{0})
	if err != nil {
		return "Open|Filtered"
	}

	// 4. Listen for service response or ICMP errors.
	buffer := make([]byte, 512)
	n, err := conn.Read(buffer)

	if err != nil {
		// Check if it's a timeout error using net.Error interface.
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return "Open|Filtered"
		}
		// Any other error (like connection refused from ICMP) means port is closed.
		return "Closed"
	}

	// 5. If we received data, port is open and responding.
	if n > 0 {
		return "Open"
	}

	return "Open|Filtered"
}

// InitUdpScan initializes resources needed for UDP scanning.
// Validates that pcap library is available and privileges are sufficient.
// Returns error if UDP scan prerequisites are not met.
func InitUdpScan() error {
	// Attempt to list network devices (requires elevated privileges).
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return fmt.Errorf("UDP scan requires root/administrator privileges and libpcap: %v", err)
	}

	if len(devices) == 0 {
		return fmt.Errorf("no network devices found for UDP scan")
	}

	return nil
}
