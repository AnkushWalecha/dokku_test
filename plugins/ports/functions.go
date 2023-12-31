package ports

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/dokku/dokku/plugins/common"
	"github.com/dokku/dokku/plugins/config"
	"github.com/ryanuber/columnize"
)

func addPortMaps(appName string, portMaps []PortMap) error {
	allPortMaps := getPortMaps(appName)
	allPortMaps = append(allPortMaps, portMaps...)

	return setPortMaps(appName, allPortMaps)
}

func clearPorts(appName string) error {
	if err := common.PropertyDelete("ports", appName, "map"); err != nil {
		return err
	}

	return common.PropertyDelete("ports", appName, "map-detected")
}

func doesCertExist(appName string) bool {
	certsExists, _ := common.PlugnTriggerOutputAsString("certs-exists", []string{appName}...)
	if certsExists == "true" {
		return true
	}

	certsForce, _ := common.PlugnTriggerOutputAsString("certs-force", []string{appName}...)
	return certsForce == "true"
}

func filterAppPortMaps(appName string, scheme string, hostPort int) []PortMap {
	var filteredPortMaps []PortMap
	for _, portMap := range getPortMaps(appName) {
		if portMap.Scheme == scheme && portMap.HostPort == hostPort {
			filteredPortMaps = append(filteredPortMaps, portMap)
		}
	}

	return filteredPortMaps
}

func getAvailablePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0
	}

	for {
		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return 0
		}
		defer l.Close()

		port := l.Addr().(*net.TCPAddr).Port
		if port >= 1025 && port <= 65535 {
			return port
		}
	}
}

func getComputedProxyPort(appName string) int {
	port := getProxyPort(appName)
	if port == 0 {
		port = getGlobalProxyPort()
	}

	return port
}

func getComputedProxySSLPort(appName string) int {
	port := getProxySSLPort(appName)
	if port == 0 {
		port = getGlobalProxySSLPort()
	}

	return port
}

func getDetectedPortMaps(appName string) []PortMap {
	basePort := getComputedProxyPort(appName)
	if basePort == 0 {
		basePort = 80
	}
	defaultMapping := []PortMap{
		{
			ContainerPort: 5000,
			HostPort:      basePort,
			Scheme:        "http",
		},
	}

	portMaps := []PortMap{}
	value, err := common.PropertyListGet("ports", appName, "map-detected")
	if err == nil {
		portMaps, _ = parsePortMapString(strings.Join(value, " "))
	}

	if len(portMaps) == 0 {
		portMaps = defaultMapping
	}

	if doesCertExist(appName) {
		setSSLPort := false
		baseSSLPort := getComputedProxySSLPort(appName)
		if baseSSLPort == 0 {
			baseSSLPort = 443
		}

		for _, portMap := range portMaps {
			if portMap.Scheme != "http" || portMap.HostPort != 80 {
				continue
			}

			setSSLPort = true
			portMaps = append(portMaps, PortMap{
				ContainerPort: portMap.ContainerPort,
				HostPort:      baseSSLPort,
				Scheme:        "https",
			})
		}

		if !setSSLPort {
			for i, portMap := range portMaps {
				if portMap.Scheme != "http" {
					continue
				}

				portMaps[i].Scheme = "https"
			}
		}
	}

	return portMaps
}

func getGlobalProxyPort() int {
	port := 0
	b, _ := common.PlugnTriggerOutput("config-get-global", []string{"DOKKU_PROXY_PORT"}...)
	if intVar, err := strconv.Atoi(strings.TrimSpace(string(b[:]))); err == nil {
		port = intVar
	}

	return port
}

func getGlobalProxySSLPort() int {
	port := 0
	b, _ := common.PlugnTriggerOutput("config-get-global", []string{"DOKKU_PROXY_SSL_PORT"}...)
	if intVar, err := strconv.Atoi(strings.TrimSpace(string(b[:]))); err == nil {
		port = intVar
	}

	return port
}

func getPortMaps(appName string) []PortMap {
	value, err := common.PropertyListGet("ports", appName, "map")
	if err != nil {
		return []PortMap{}
	}

	portMaps, _ := parsePortMapString(strings.Join(value, " "))
	return portMaps
}

func getProxyPort(appName string) int {
	port := 0
	b, _ := common.PlugnTriggerOutput("config-get", []string{appName, "DOKKU_PROXY_PORT"}...)
	if intVar, err := strconv.Atoi(strings.TrimSpace(string(b[:]))); err == nil {
		port = intVar
	}

	return port
}

func getProxySSLPort(appName string) int {
	port := 0
	b, _ := common.PlugnTriggerOutput("config-get", []string{appName, "DOKKU_PROXY_SSL_PORT"}...)
	if intVar, err := strconv.Atoi(strings.TrimSpace(string(b[:]))); err == nil {
		port = intVar
	}

	return port
}

func initializeProxyPort(appName string) error {
	port := getProxyPort(appName)
	if port != 0 {
		return nil
	}

	if isAppVhostEnabled(appName) {
		port = getGlobalProxyPort()
	} else {
		common.LogInfo1("No port set, setting to random open high port")
		port = getAvailablePort()
		common.LogInfo1(fmt.Sprintf("Random port %d", port))
	}

	if port == 0 {
		port = 80
	}

	if err := setProxyPort(appName, port); err != nil {
		return err
	}
	return nil
}

func initializeProxySSLPort(appName string) error {
	port := getProxySSLPort(appName)
	if port != 0 {
		return nil
	}

	if !doesCertExist(appName) {
		return nil
	}

	port = getGlobalProxySSLPort()
	if port == 0 {
		port = 443
	}

	if !isAppVhostEnabled(appName) {
		common.LogInfo1("No ssl port set, setting to random open high port")
		port = getAvailablePort()
	}

	if err := setProxySSLPort(appName, port); err != nil {
		return err
	}

	return nil
}

func inRange(value int, min int, max int) bool {
	return min < value && value < max
}

func isAppVhostEnabled(appName string) bool {
	if err := common.PlugnTrigger("domains-vhost-enabled", []string{appName}...); err != nil {
		return false
	}
	return true
}

func listAppPortMaps(appName string) error {
	portMaps := getPortMaps(appName)

	if len(portMaps) == 0 {
		return errors.New("No port mappings configured for app")
	}

	var lines []string
	if os.Getenv("DOKKU_QUIET_OUTPUT") == "" {
		lines = append(lines, "-----> scheme:host port:container port")
	}

	for _, portMap := range portMaps {
		lines = append(lines, portMap.String())
	}

	sort.Strings(lines)
	common.LogInfo1Quiet(fmt.Sprintf("Port mappings for %s", appName))
	config := columnize.DefaultConfig()
	config.Delim = ":"
	config.Prefix = "    "
	config.Empty = ""
	fmt.Println(columnize.Format(lines, config))
	return nil
}

func parsePortMapString(stringPortMap string) ([]PortMap, error) {
	var portMaps []PortMap

	for _, v := range strings.Split(strings.TrimSpace(stringPortMap), " ") {
		parts := strings.SplitN(v, ":", 3)
		if len(parts) == 1 {
			hostPort, err := strconv.Atoi(v)
			if err != nil {
				return portMaps, fmt.Errorf("Invalid port map %s [err=%s]", v, err.Error())
			}

			if !inRange(hostPort, 0, 65536) {
				return portMaps, fmt.Errorf("Invalid port map %s [hostPort=%d]", v, hostPort)
			}

			portMaps = append(portMaps, PortMap{
				HostPort: hostPort,
				Scheme:   "__internal__",
			})
			continue
		}

		if len(parts) != 3 {
			return portMaps, fmt.Errorf("Invalid port map %s [len=%d]", v, len(parts))
		}

		hostPort, err := strconv.Atoi(parts[1])
		if err != nil {
			return portMaps, fmt.Errorf("Invalid port map %s [err=%s]", v, err.Error())
		}

		containerPort, err := strconv.Atoi(parts[2])
		if err != nil {
			return portMaps, fmt.Errorf("Invalid port map %s [err=%s]", v, err.Error())
		}

		if !inRange(hostPort, 0, 65536) {
			return portMaps, fmt.Errorf("Invalid port map %s [hostPort=%d]", v, hostPort)
		}

		if !inRange(containerPort, 0, 65536) {
			return portMaps, fmt.Errorf("Invalid port map %s [containerPort=%d]", v, containerPort)
		}

		portMaps = append(portMaps, PortMap{
			ContainerPort: containerPort,
			HostPort:      hostPort,
			Scheme:        parts[0],
		})
	}

	return uniquePortMaps(portMaps), nil
}

func removePortMaps(appName string, portMaps []PortMap) error {
	toRemove := map[string]bool{}
	toRemoveByPort := map[int]bool{}

	for _, portMap := range portMaps {
		if portMap.AllowsPersistence() {
			toRemoveByPort[portMap.HostPort] = true
			continue
		}
		toRemove[portMap.String()] = true
	}

	var toSet []PortMap
	for _, portMap := range getPortMaps(appName) {
		if toRemove[portMap.String()] {
			continue
		}

		if toRemoveByPort[portMap.HostPort] {
			continue
		}

		toSet = append(toSet, portMap)
	}

	if len(toSet) == 0 {
		return common.PropertyDelete("ports", appName, "map")
	}

	return setPortMaps(appName, toSet)
}

func setPortMaps(appName string, portMaps []PortMap) error {
	var value []string
	for _, portMap := range uniquePortMaps(portMaps) {
		if portMap.AllowsPersistence() {
			continue
		}

		value = append(value, portMap.String())
	}

	sort.Strings(value)
	return common.PropertyListWrite("ports", appName, "map", value)
}

func setProxyPort(appName string, port int) error {
	return common.EnvWrap(func() error {
		entries := map[string]string{
			"DOKKU_PROXY_PORT": fmt.Sprint(port),
		}
		return config.SetMany(appName, entries, false)
	}, map[string]string{"DOKKU_QUIET_OUTPUT": "1"})
}

func setProxySSLPort(appName string, port int) error {
	return common.EnvWrap(func() error {
		entries := map[string]string{
			"DOKKU_PROXY_SSL_PORT": fmt.Sprint(port),
		}
		return config.SetMany(appName, entries, false)
	}, map[string]string{"DOKKU_QUIET_OUTPUT": "1"})
}

func uniquePortMaps(portMaps []PortMap) []PortMap {
	var unique []PortMap
	existingPortMaps := map[string]bool{}

	for _, portMap := range portMaps {
		if existingPortMaps[portMap.String()] {
			continue
		}

		existingPortMaps[portMap.String()] = true
		unique = append(unique, portMap)
	}

	return unique
}
