package proxy

import (
	"github.com/dokku/dokku/plugins/common"
)

// ReportSingleApp is an internal function that displays the proxy report for one or more apps
func ReportSingleApp(appName string, format string, infoFlag string) error {
	if err := common.VerifyAppName(appName); err != nil {
		return err
	}

	flags := map[string]common.ReportFunc{
		"--proxy-enabled": reportEnabled,
		"--proxy-type":    reportType,
	}

	flagKeys := []string{}
	for flagKey := range flags {
		flagKeys = append(flagKeys, flagKey)
	}

	trimPrefix := false
	uppercaseFirstCharacter := true
	infoFlags := common.CollectReport(appName, infoFlag, flags)
	return common.ReportSingleApp("proxy", appName, infoFlag, infoFlags, flagKeys, format, trimPrefix, uppercaseFirstCharacter)
}

func reportEnabled(appName string) string {
	proxyEnabled := "false"
	if IsAppProxyEnabled(appName) {
		proxyEnabled = "true"
	}

	return proxyEnabled
}

func reportType(appName string) string {
	return getAppProxyType(appName)
}
