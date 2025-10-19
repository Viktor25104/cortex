package scanner

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// TCPSynWorker processes scan jobs using TCP SYN scan (half-open/stealth scan).
// Sends SYN packet and analyzes the response (SYN-ACK or RST) without completing
// the three-way handshake, making it harder to detect than TCP Connect scan.
// Requires elevated privileges (root/administrator) for raw socket access.
// Note: cache parameter is unused as SYN scan operates at packet level and cannot
// perform application-layer service detection.
func TCPSynWorker(jobs <-chan ScanJob, results chan<- ScanResult, cache *ProbeCache, wg *sync.WaitGroup) {
	_ = cache // Unused: SYN scanning operates at network layer only
	for job := range jobs {
		state := performSynScan(job.Host, job.Port)
		result := ScanResult{Host: job.Host, Port: job.Port, State: state}
		results <- result
		wg.Done()
	}
}

// performSynScan executes a TCP SYN scan on a single target port.
// Constructs and sends a raw TCP SYN packet, then analyzes the response
// to determine port state. Returns:
// - "Open": SYN-ACK received (port accepting connections)
// - "Closed": RST received (port actively refusing connections)
// - "Filtered": Timeout or local errors (cannot determine state)
func performSynScan(host string, port int) string {
	// Find all available network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		return "Filtered" // Local error - cannot determine port state
	}

	var srcIP net.IP
	var device *net.Interface

	// Select a suitable network interface and source IP address
	// Criteria: interface must be up, not loopback, and have an IPv4 address
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
		return "Filtered" // Local error - no suitable interface found
	}

	// Resolve target hostname to IP address
	dstIPs, err := net.LookupIP(host)
	if err != nil {
		return "Filtered" // DNS resolution failed - cannot determine port state
	}

	dstIP := dstIPs[0].To4()
	if dstIP == nil {
		return "Filtered" // IPv6 or invalid IP - not supported
	}

	// Open packet capture handle for raw packet transmission and reception
	handle, err := pcap.OpenLive(device.Name, 65535, false, 2*time.Second)
	if err != nil {
		return "Filtered" // Local error - cannot open pcap handle
	}
	defer handle.Close()

	// Construct TCP SYN packet with randomized source port
	srcPort := uint16(rand.Intn(65535-1024) + 1024) // Use ephemeral port range
	dstPort := uint16(port)

	// Update BPF filter to include destination port for precise packet capture
	// This prevents false positives from unrelated traffic
	filter := fmt.Sprintf("tcp and src host %s and src port %d and dst host %s and dst port %d",
		dstIP.String(), port, srcIP.String(), srcPort)
	if err := handle.SetBPFFilter(filter); err != nil {
		return "Filtered" // Local error - cannot set BPF filter
	}

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
		Seq:     rand.Uint32(),
	}

	// Set network layer for proper TCP checksum calculation
	_ = tcpLayer.SetNetworkLayerForChecksum(ipLayer)

	// Serialize packet layers into transmittable byte buffer
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(buffer, opts, ipLayer, tcpLayer); err != nil {
		return "Filtered" // Local error - cannot serialize packet
	}

	// Transmit the SYN packet to the target
	if err := handle.WritePacketData(buffer.Bytes()); err != nil {
		return "Filtered" // Local error - cannot send packet
	}

	// Listen for TCP response with timeout
	timeout := time.After(2 * time.Second)
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	for {
		select {
		case packet := <-packetSource.Packets():
			if packet == nil {
				return "Filtered" // No packet received - ambiguous state
			}

			// Extract TCP layer and analyze flags
			if tcpPacket, ok := packet.Layer(layers.LayerTypeTCP).(*layers.TCP); ok {
				if tcpPacket.SYN && tcpPacket.ACK {
					return "Open" // SYN-ACK indicates open port
				}
				if tcpPacket.RST {
					return "Closed" // RST indicates closed port
				}
			}

		case <-timeout:
			return "Filtered" // Timeout - packets likely dropped by firewall
		}
	}
}

// InitSynScan validates that the system meets prerequisites for SYN scanning.
// Checks for libpcap availability and verifies elevated privileges by attempting
// to enumerate network devices. Returns error if requirements are not satisfied.
func InitSynScan() error {
	// Enumerate network devices (requires elevated privileges)
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return fmt.Errorf("SYN scan requires root/administrator privileges and libpcap: %v", err)
	}

	if len(devices) == 0 {
		return fmt.Errorf("no network devices found for SYN scan")
	}

	return nil
}
