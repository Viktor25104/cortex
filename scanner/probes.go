package scanner

import (
	"bufio"
	"errors"
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

// ParseError stores information about a parsing error on a specific line.
type ParseError struct {
	LineNumber int
	Message    string
}

// LoadStats stores statistics about the probe loading process.
type LoadStats struct {
	TotalLines int
	ProbeCount int
	MatchCount int
	ErrorLines []ParseError
}

// LoadProbes reads and parses probe definitions from a file.
// Returns probes slice, detailed loading statistics, and error if file cannot be read.
func LoadProbes(filePath string) ([]Probe, LoadStats, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, LoadStats{}, fmt.Errorf("cannot open file %s: %w", filePath, err)
	}
	defer file.Close()

	var probes []Probe
	var currentProbe *Probe // Use pointer for convenience
	stats := LoadStats{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		stats.TotalLines++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "Probe") {
			// If there was a previous probe, add it to the list
			if currentProbe != nil {
				probes = append(probes, *currentProbe)
			}
			probe, err := parseProbe(line)
			if err != nil {
				stats.ErrorLines = append(stats.ErrorLines, ParseError{stats.TotalLines, err.Error()})
				currentProbe = nil
				continue
			}
			currentProbe = &probe
			stats.ProbeCount++

		} else if strings.HasPrefix(line, "match ") {
			if currentProbe == nil {
				stats.ErrorLines = append(stats.ErrorLines, ParseError{stats.TotalLines, "match found without preceding Probe"})
				continue
			}
			match, err := parseMatch(line)
			if err != nil {
				// Check if this is an unsupported regex (not a real error)
				var unsupportedErr *UnsupportedRegexError
				if errors.As(err, &unsupportedErr) {
					// Silently skip unsupported Perl regex patterns
					// These are valid in nmap but not supported by Go's RE2 engine
					continue
				}
				// Real parsing error - log it
				stats.ErrorLines = append(stats.ErrorLines, ParseError{stats.TotalLines, fmt.Sprintf("match parse error: %v", err)})
				continue
			}
			currentProbe.Matches = append(currentProbe.Matches, match)
			stats.MatchCount++

		} else if isKnownDirective(line) {
			// Known directives that we currently ignore (not counted as errors)
			// These directives are valid but not used in our implementation:
			// - softmatch: Fuzzy service matching (we use only strict 'match')
			// - ports/sslports: Port hints (we scan all specified ports)
			// - rarity: Probe rarity level (we try all probes sequentially)
			// - fallback: Fallback probe name (not implemented)
			// - Exclude: Port exclusion (not implemented)
			// - totalwaitms/tcpwrappedms: Global timeouts (we use fixed timeouts)
			continue
		} else {
			stats.ErrorLines = append(stats.ErrorLines, ParseError{stats.TotalLines, "Unknown line format"})
		}
	}

	// Add the last probe to the list
	if currentProbe != nil {
		probes = append(probes, *currentProbe)
	}

	if err := scanner.Err(); err != nil {
		return nil, stats, fmt.Errorf("error reading file: %w", err)
	}

	return probes, stats, nil
}

// isKnownDirective checks if a line starts with a known nmap-service-probes directive
// that we intentionally ignore (not an error, just not implemented).
func isKnownDirective(line string) bool {
	knownDirectives := []string{
		"softmatch",       // Fuzzy matching rules
		"ports",           // Port hints for this probe
		"sslports",        // SSL port hints
		"rarity",          // Probe rarity (1-9, higher = more rare)
		"fallback",        // Fallback probe name
		"Exclude",         // Exclude specific ports
		"totalwaitms",     // Global wait timeout
		"tcpwrappedms",    // TCP wrapped detection timeout
	}

	for _, directive := range knownDirectives {
		if strings.HasPrefix(line, directive) {
			return true
		}
	}
	return false
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
// Also handles optional attributes after closing | (e.g., q|..| no-payload)
func parseProbeData(dataStr string) ([]byte, error) {
	// Check minimum format q|...|
	if len(dataStr) < 3 || dataStr[0] != 'q' || dataStr[1] != '|' {
		return nil, fmt.Errorf("probe data must be in format q|...|")
	}

	// Find the closing | delimiter
	closingPipeIndex := strings.LastIndex(dataStr[2:], "|")
	if closingPipeIndex == -1 {
		return nil, fmt.Errorf("probe data must be in format q|...|")
	}
	closingPipeIndex += 2 // Adjust for the offset

	// Extract content between q| and the first closing |
	// Ignore anything after the closing pipe (e.g., "no-payload", "source=500")
	content := dataStr[2:closingPipeIndex]

	// Normalize escape sequences for Go compatibility
	content = normalizeEscapeSequences(content)

	// Escape any unescaped double quotes in the content
	// This is necessary because we'll wrap the content in quotes for strconv.Unquote
	content = escapeInternalQuotes(content)

	// Add quotes and use strconv.Unquote to handle escape sequences
	quotedContent := "\"" + content + "\""
	unquoted, err := strconv.Unquote(quotedContent)
	if err != nil {
		return nil, fmt.Errorf("cannot unquote probe data: %w", err)
	}

	return []byte(unquoted), nil
}

// escapeInternalQuotes escapes any unescaped double quotes in the string.
// This is needed before wrapping content in quotes for strconv.Unquote.
// We need to be careful not to escape already-escaped quotes (\\").
func escapeInternalQuotes(s string) string {
	var result strings.Builder
	result.Grow(len(s) + 10)

	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			// Check if this quote is already escaped
			if i > 0 && s[i-1] == '\\' {
				// Already escaped, keep as is
				result.WriteByte(s[i])
			} else {
				// Unescaped quote - escape it
				result.WriteString(`\"`)
			}
		} else {
			result.WriteByte(s[i])
		}
	}

	return result.String()
}

// normalizeEscapeSequences normalizes escape sequences from nmap format to Go format.
// Handles:
// - \0 -> \x00 (octal null byte to hex when not part of longer octal sequence)
// - \xAB -> \xab (uppercase hex to lowercase - actually not needed but kept for clarity)
func normalizeEscapeSequences(s string) string {
	var result strings.Builder
	result.Grow(len(s) + 10) // Extra space for replacements

	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			nextChar := s[i+1]

			// Handle \0 -> \x00 (convert short octal to hex)
			// According to Go's strconv.Unquote, octal sequences are:
			// - \0 through \7 (single digit)
			// - \00 through \77 (two digits)
			// - \000 through \377 (three digits)
			// We need to convert standalone \0 to \x00 when it's NOT part of multi-digit octal
			if nextChar == '0' {
				// Check what comes after \0
				if i+2 >= len(s) {
					// \0 at end of string - convert to \x00
					result.WriteString("\\x00")
					i += 1
					continue
				}

				thirdChar := s[i+2]

				// If the third character is an octal digit (0-7), this could be \00X or \0XX
				// To determine, we check if it forms a valid 2 or 3 digit octal:
				// - If third char is 0-7 AND fourth char (if exists) is 0-7: it's \0XX (3-digit octal) - keep as is
				// - If third char is 0-7 AND fourth char is NOT 0-7 or doesn't exist: it's \0 followed by \X (separate) - convert \0
				if isOctalDigit(thirdChar) {
					// Check if there's a fourth character and if it's also octal
					if i+3 < len(s) && isOctalDigit(s[i+3]) {
						// This is a 3-digit octal sequence \0XX - keep as is
						result.WriteByte(s[i])
						continue
					}
					// Otherwise, \0 is standalone, followed by separate escape or character
					result.WriteString("\\x00")
					i += 1
					continue
				}

				// Third char is NOT an octal digit (not 0-7) - \0 is standalone
				result.WriteString("\\x00")
				i += 1
				continue
			}

			// Handle \xXX (uppercase hex digits)
			if nextChar == 'x' && i+3 < len(s) {
				result.WriteString("\\x")
				result.WriteByte(toLowerHexDigit(s[i+2]))
				result.WriteByte(toLowerHexDigit(s[i+3]))
				i += 3
				continue
			}
		}

		result.WriteByte(s[i])
	}

	return result.String()
}

// isOctalDigit checks if a character is an octal digit (0-7)
func isOctalDigit(c byte) bool {
	return c >= '0' && c <= '7'
}

// toLowerHexDigit converts a hex digit to lowercase (A-F -> a-f, 0-9 unchanged)
func toLowerHexDigit(c byte) byte {
	if c >= 'A' && c <= 'F' {
		return c + ('a' - 'A')
	}
	return c
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

	if len(patternStr) < 2 || patternStr[0] != 'm' {
		return Match{}, fmt.Errorf("invalid match pattern format: missing 'm'")
	}

	// Dynamically determine which character is used as separator
	separator := string(patternStr[1])

	// Split the string using this separator
	// We expect 3 parts: empty string before separator, pattern, flags and version after
	patternParts := strings.SplitN(patternStr[2:], separator, 2)
	if len(patternParts) < 2 {
		return Match{}, fmt.Errorf("invalid match pattern format: could not split pattern and flags using separator '%s'", separator)
	}

	pattern := patternParts[0]
	flagsAndVersion := patternParts[1]

	// Build regex with flags if present
	regexStr := pattern
	if strings.Contains(flagsAndVersion, "i") {
		regexStr = "(?i)" + regexStr
	}
	if strings.Contains(flagsAndVersion, "s") {
		regexStr = "(?s)" + regexStr
	}

	// Check if pattern contains unsupported Perl regex features
	if containsUnsupportedRegex(regexStr) {
		return Match{}, &UnsupportedRegexError{Pattern: regexStr}
	}

	// Try to compile the regex
	regex, err := regexp.Compile(regexStr)
	if err != nil {
		// Check if this is a Go RE2 limitation (e.g., repeat count > 1000)
		// These are valid regex patterns but not supported by Go's engine
		if strings.Contains(err.Error(), "invalid repeat count") {
			return Match{}, &UnsupportedRegexError{Pattern: regexStr}
		}
		return Match{}, fmt.Errorf("cannot compile regex '%s': %w", regexStr, err)
	}

	// TODO: Parse version information (p/v/i/o) in future implementation
	return Match{
		ServiceName: serviceName,
		Pattern:     regex,
		VersionInfo: make(map[string]string),
	}, nil
}

// UnsupportedRegexError indicates a Perl regex feature not supported by Go
type UnsupportedRegexError struct {
	Pattern string
}

func (e *UnsupportedRegexError) Error() string {
	return fmt.Sprintf("unsupported Perl regex (lookahead/lookbehind/backreference)")
}

// containsUnsupportedRegex checks if pattern contains Perl regex features not supported by Go
func containsUnsupportedRegex(pattern string) bool {
	// Check for lookahead/lookbehind assertions
	unsupportedPatterns := []string{
		`(?!`,  // Negative lookahead
		`(?=`,  // Positive lookahead
		`(?<=`, // Positive lookbehind
		`(?<!`, // Negative lookbehind
		`\1`,   // Backreference to group 1
		`\2`,   // Backreference to group 2
		`\3`,   // Backreference to group 3
		`\4`,   // Backreference to group 4
		`\5`,   // Backreference to group 5
		`\6`,   // Backreference to group 6
		`\7`,   // Backreference to group 7
		`\8`,   // Backreference to group 8
		`\9`,   // Backreference to group 9
	}

	for _, unsupported := range unsupportedPatterns {
		if strings.Contains(pattern, unsupported) {
			return true
		}
	}
	return false
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
