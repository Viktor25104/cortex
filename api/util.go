package api

import (
	"fmt"
	"strconv"
	"strings"
)

func parsePortRange(portRange string) (int, int, error) {
	parts := strings.Split(portRange, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid port range format. Use startPort-endPort")
	}

	startPort, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("start port is not a number: %s", parts[0])
	}

	endPort, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("end port is not a number: %s", parts[1])
	}

	if startPort < 0 || startPort > 65535 || endPort < 0 || endPort > 65535 {
		return 0, 0, fmt.Errorf("ports must be within 0-65535 range")
	}

	if startPort > endPort {
		return 0, 0, fmt.Errorf("start port must be less than or equal to end port")
	}

	return startPort, endPort, nil
}
