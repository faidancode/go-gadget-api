package platform

import (
	"strings"
)

type ClientType string

const (
	WebAdmin    ClientType = "web-admin"
	WebCustomer ClientType = "web-customer"
	Mobile      ClientType = "mobile"
)

func ResolveClientType(clientHeader, userAgent string) ClientType {
	header := strings.ToLower(clientHeader)

	// 1. Explicit check dari custom header
	switch header {
	case "mobile":
		return Mobile
	case "web-admin":
		return WebAdmin
	case "web-customer":
		return WebCustomer
	}

	// 2. Auto-detect via User-Agent
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "expo") ||
		strings.Contains(ua, "reactnative") ||
		strings.Contains(ua, "react-native") ||
		strings.Contains(ua, "okhttp") {
		return Mobile
	}

	return WebCustomer
}

func IsWebClient(ct ClientType) bool {
	return ct == WebAdmin || ct == WebCustomer
}
