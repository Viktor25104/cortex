package scanner

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Probe represents a single probe for service detection.
type Probe struct {
	Protocol string  // TCP or UDP
	Name     string  // Probe name, e.g. "GetRequest"
	Data     []byte  // Data to send to the server
	Matches  []Match // List of patterns to match in response
}

// Match represents a single service detection rule.
type Match struct {
	ServiceName string            // Service name, e.g. "http"
	Pattern     *regexp.Regexp    // Compiled regex pattern to match
	VersionInfo map[string]string // Additional version information
}

// LoadProbes reads and parses the nmap-service-probes file.
func LoadProbes(filePath string) ([]Probe, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file %s: %w", filePath, err)
	}
	defer file.Close()

	var probes []Probe
	var currentProbe Probe
	scanner := bufio.NewScanner(file)
	hasCurrentProbe := false

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Determine line type
		if strings.HasPrefix(line, "Probe") {
			// Parse Probe line
			probe, err := parseProbe(line)
			if err != nil {
				fmt.Printf("Warning at line %d: %v\n", lineNum, err)
				continue
			}

			// If there was a previous probe, add it
			if hasCurrentProbe {
				probes = append(probes, currentProbe)
			}

			currentProbe = probe
			hasCurrentProbe = true
			fmt.Printf("[DEBUG] Loaded Probe: %s %s\n", probe.Protocol, probe.Name)

		} else if strings.HasPrefix(line, "match") {
			// Parse match line and add to current probe
			if !hasCurrentProbe {
				fmt.Printf("Warning at line %d: match found without preceding Probe\n", lineNum)
				continue
			}

			match, err := parseMatch(line)
			if err != nil {
				fmt.Printf("Warning at line %d: %v\n", lineNum, err)
				continue
			}

			currentProbe.Matches = append(currentProbe.Matches, match)
			fmt.Printf("[DEBUG] Added Match for service: %s\n", match.ServiceName)

		} else {
			// Unknown line format
			fmt.Printf("Info at line %d: Unknown line format (skipped)\n", lineNum)
		}
	}

	// Add last probe if exists
	if hasCurrentProbe {
		probes = append(probes, currentProbe)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	fmt.Printf("[DEBUG] Total probes loaded: %d\n", len(probes))
	return probes, nil
}

// parseProbe parses a line like:
// Probe TCP GetRequest q|GET / HTTP/1.0\r\n\r\n|
func parseProbe(line string) (Probe, error) {
	line = strings.TrimPrefix(line, "Probe ")

	// Split into 3 parts: Protocol, Name, Data
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return Probe{}, fmt.Errorf("invalid Probe format")
	}

	protocol := parts[0]
	name := parts[1]
	dataStr := parts[2]

	data, err := parseProbeData(dataStr)
	if err != nil {
		return Probe{}, fmt.Errorf("cannot parse probe data: %w", err)
	}

	return Probe{
		Protocol: protocol,
		Name:     name,
		Data:     data,
		Matches:  []Match{},
	}, nil
}

// parseProbeData converts a string in format q|...|  to a byte slice.
// Supports escape sequences like \n, \r, \x00, etc.
func parseProbeData(dataStr string) ([]byte, error) {
	// Check format q|...|
	if len(dataStr) < 3 || dataStr[0] != 'q' || dataStr[1] != '|' || dataStr[len(dataStr)-1] != '|' {
		return nil, fmt.Errorf("probe data must be in format q|...|")
	}

	// Extract content between q| and |
	content := dataStr[2 : len(dataStr)-1]

	// Add quotes and use strconv.Unquote to handle escape sequences
	quotedContent := "\"" + content + "\""
	unquoted, err := strconv.Unquote(quotedContent)
	if err != nil {
		return nil, fmt.Errorf("cannot unquote probe data: %w", err)
	}

	return []byte(unquoted), nil
}

// parseMatch parses a line like:
// match service m|pattern|flags
func parseMatch(line string) (Match, error) {
	line = strings.TrimPrefix(line, "match ")
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return Match{}, fmt.Errorf("invalid match format")
	}

	serviceName := parts[0]
	patternStr := parts[1]

	// Split m|pattern|flags into 3 parts
	patternParts := strings.SplitN(patternStr, "|", 3)
	if len(patternParts) < 3 || patternParts[0] != "m" {
		return Match{}, fmt.Errorf("invalid match pattern format: %s", patternStr)
	}

	pattern := patternParts[1]
	flags := patternParts[2]

	regexStr := pattern
	if strings.Contains(flags, "i") {
		regexStr = "(?i)" + regexStr
	}
	if strings.Contains(flags, "s") {
		regexStr = "(?s)" + regexStr
	}

	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return Match{}, fmt.Errorf("cannot compile regex: %w", err)
	}

	return Match{
		ServiceName: serviceName,
		Pattern:     regex,
		VersionInfo: make(map[string]string),
	}, nil
}

// ProbeCache caches loaded probes for fast access
type ProbeCache struct {
	allProbes   []Probe
	tcpProbes   []Probe
	udpProbes   []Probe
	probeLookup map[string][]Probe // by probe name
}

// NewProbeCache creates and initializes probe cache
func NewProbeCache(probes []Probe) *ProbeCache {
	cache := &ProbeCache{
		allProbes:   probes,
		probeLookup: make(map[string][]Probe),
	}

	for _, probe := range probes {
		if probe.Protocol == "TCP" {
			cache.tcpProbes = append(cache.tcpProbes, probe)
		} else if probe.Protocol == "UDP" {
			cache.udpProbes = append(cache.udpProbes, probe)
		}
		cache.probeLookup[probe.Name] = append(cache.probeLookup[probe.Name], probe)
	}

	return cache
}

// GetTCPProbes returns all TCP probes
func (pc *ProbeCache) GetTCPProbes() []Probe {
	return pc.tcpProbes
}

// GetUDPProbes returns all UDP probes
func (pc *ProbeCache) GetUDPProbes() []Probe {
	return pc.udpProbes
}

// GetProbeByName returns probe by name
func (pc *ProbeCache) GetProbeByName(name string) ([]Probe, bool) {
	probes, exists := pc.probeLookup[name]
	return probes, exists
}
