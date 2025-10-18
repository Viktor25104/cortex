package scanner

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// TCPSynWorker processes scan jobs using SYN Scan (stealth mode).
// Sends SYN packet and detects response without completing handshake.
// Requires root/administrator privileges on Unix-like systems.
func TCPSynWorker(jobs <-chan ScanJob, results chan<- ScanResult) {
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
