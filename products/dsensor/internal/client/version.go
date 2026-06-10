package client

import (
	"encoding/json"
	"fmt"
)

// PackageInfoRet is the response from /api/meta/package/info.
type PackageInfoRet struct {
	Current *CurrentVersion `json:"current,omitempty"`
}

// CurrentVersion contains version information.
type CurrentVersion struct {
	Manager string `json:"manager,omitempty"`
	Agent   string `json:"agent,omitempty"`
}

// VersionInfo is the resolved version information.
type VersionInfo struct {
	ManagerVersion string `json:"manager_version"`
	AgentVersion   string `json:"agent_version"`
}

// GetVersion fetches version info from the server.
func (c *Client) GetVersion() (*VersionInfo, error) {
	_, apiResp, err := c.DoAndParse("/api/meta/package/info", []byte("{}"))
	if err != nil {
		return nil, fmt.Errorf("获取版本信息失败: %w", err)
	}

	var pkg PackageInfoRet
	if err := json.Unmarshal(apiResp.Data, &pkg); err != nil {
		return nil, fmt.Errorf("解析版本信息失败: %w", err)
	}

	v := &VersionInfo{}
	if pkg.Current != nil {
		v.ManagerVersion = pkg.Current.Manager
		v.AgentVersion = pkg.Current.Agent
	}

	return v, nil
}
