package proxy

import (
	"github.com/dokku/dokku/plugins/common"
	"github.com/dokku/dokku/plugins/config"
)

// RunInSerial is the default value for whether to run a command in parallel or not
// and defaults to -1 (false)
const RunInSerial = 0

// BuildConfig rebuilds the proxy config for the specified app
func BuildConfig(appName string) error {
	return common.PlugnTrigger("proxy-build-config", []string{appName}...)
}

// ClearConfig clears the proxy config for the specified app
func ClearConfig(appName string) error {
	return common.PlugnTrigger("proxy-clear-config", []string{appName}...)
}

// Disable disables proxy implementations for the specified app
func Disable(appName string) error {
	if !IsAppProxyEnabled(appName) {
		common.LogInfo1("Proxy is already disable for app")
		return nil
	}

	common.LogInfo1("Disabling proxy for app")
	entries := map[string]string{
		"DOKKU_DISABLE_PROXY": "1",
	}

	if err := config.SetMany(appName, entries, false); err != nil {
		return err
	}

	return common.PlugnTrigger("proxy-disable", []string{appName}...)
}

// Enable enables proxy implementations for the specified app
func Enable(appName string) error {
	if IsAppProxyEnabled(appName) {
		common.LogInfo1("Proxy is already enabled for app")
		return nil
	}

	common.LogInfo1("Enabling proxy for app")
	keys := []string{"DOKKU_DISABLE_PROXY"}
	if err := config.UnsetMany(appName, keys, false); err != nil {
		return err
	}

	return common.PlugnTrigger("proxy-enable", []string{appName}...)
}

// IsAppProxyEnabled returns true if proxy is enabled; otherwise return false
func IsAppProxyEnabled(appName string) bool {
	proxyEnabled := true
	disableProxy := config.GetWithDefault(appName, "DOKKU_DISABLE_PROXY", "")
	if disableProxy != "" {
		proxyEnabled = false
	}
	return proxyEnabled
}
