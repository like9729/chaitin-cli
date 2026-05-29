package runtime

import (
	"fmt"
	"strings"

	safelineversion "github.com/chaitin/chaitin-cli/products/safeline/version"
)

type OperationMode string

const (
	ModeSoftwareReverseProxy        OperationMode = "Software Reverse Proxy"
	ModeSoftwareClusterReverseProxy OperationMode = "Software Cluster Reverse Proxy"
	ModeHardwareReverseProxy        OperationMode = "Hardware Reverse Proxy"
	ModeHardwareTransparentProxy    OperationMode = "Hardware Transparent Proxy"
	ModeHardwareTransparentBridging OperationMode = "Hardware Transparent Bridging"
	ModeSoftwarePortMirroring       OperationMode = "Software Port Mirroring"
	ModeHardwarePortMirroring       OperationMode = "Hardware Port Mirroring"
	ModeHardwareTrafficDetection    OperationMode = "Hardware Traffic Detection"
	ModeHardwareRouterProxy         OperationMode = "Hardware Router Proxy"
)

type VersionFamily string

const (
	Family23_01   VersionFamily = "23.01.x"
	Family25_03   VersionFamily = "25.03.x"
	FamilyUnknown VersionFamily = "unknown"
)

type Context struct {
	Version             string        `json:"version,omitempty"`
	VersionSource       string        `json:"version_source,omitempty"`
	VersionFamily       VersionFamily `json:"version_family"`
	OperationMode       OperationMode `json:"operation_mode,omitempty"`
	OperationModeSource string        `json:"operation_mode_source,omitempty"`
	Endpoint            string        `json:"endpoint,omitempty"`
	SiteCreateSupported bool          `json:"site_create_supported"`
	Warnings            []string      `json:"warnings,omitempty"`
}

func NormalizeOperationMode(raw string) (OperationMode, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	key = strings.ReplaceAll(key, "_", "-")
	switch key {
	case "software reverse proxy", "software-reverse-proxy":
		return ModeSoftwareReverseProxy, nil
	case "software cluster reverse proxy", "software-cluster-reverse-proxy":
		return ModeSoftwareClusterReverseProxy, nil
	case "hardware reverse proxy", "hardware-reverse-proxy":
		return ModeHardwareReverseProxy, nil
	case "hardware transparent proxy", "hardware-transparent-proxy":
		return ModeHardwareTransparentProxy, nil
	case "hardware transparent bridging", "hardware-transparent-bridging":
		return ModeHardwareTransparentBridging, nil
	case "software port mirroring", "software-port-mirroring":
		return ModeSoftwarePortMirroring, nil
	case "hardware port mirroring", "hardware-port-mirroring":
		return ModeHardwarePortMirroring, nil
	case "hardware traffic detection", "hardware-traffic-detection":
		return ModeHardwareTrafficDetection, nil
	case "hardware router proxy", "hardware-router-proxy":
		return ModeHardwareRouterProxy, nil
	default:
		return "", fmt.Errorf("unsupported operation mode %q", raw)
	}
}

func ClassifyVersion(raw string) VersionFamily {
	v, err := safelineversion.ParseVersion(raw)
	if err != nil {
		return FamilyUnknown
	}
	if v.Major == 23 && v.Minor == 1 {
		return Family23_01
	}
	if v.Major == 25 && v.Minor == 3 {
		return Family25_03
	}
	return FamilyUnknown
}

func EndpointForMode(mode OperationMode) (string, bool) {
	switch mode {
	case ModeSoftwareReverseProxy:
		return "/api/SoftwareReverseProxyWebsiteAPI", true
	case ModeSoftwareClusterReverseProxy:
		return "/api/SoftwareClusterReverseProxyWebsiteAPI", true
	case ModeHardwareReverseProxy:
		return "/api/HardwareReverseProxyWebsiteAPI", true
	case ModeHardwareTransparentProxy:
		return "/api/HardwareTransparentProxyWebsiteAPI", true
	case ModeHardwareTransparentBridging:
		return "/api/HardwareTransparentBridgingWebsiteAPI", true
	case ModeSoftwarePortMirroring:
		return "/api/SoftwarePortMirroringWebsiteAPI", true
	case ModeHardwarePortMirroring, ModeHardwareTrafficDetection:
		return "/api/HardwareTrafficDetectionWebsiteAPI", true
	case ModeHardwareRouterProxy:
		return "/api/HardwareReverseProxyWebsiteAPI", true
	default:
		return "", false
	}
}
