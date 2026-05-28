package runtime

import (
	"encoding/json"
	"fmt"

	"github.com/chaitin/chaitin-cli/products/safeline/pkg/client"
)

type Options struct {
	VersionOverride       string
	OperationModeOverride string
	ConfigVersion         string
	ConfigOperationMode   string
}

type remoteConfig struct {
	Version       string   `json:"version"`
	OperationMode []string `json:"operation_mode"`
}

func ResolveContext(cl *client.Client, opts Options) (Context, error) {
	ctx := Context{VersionFamily: FamilyUnknown}
	var remote remoteConfig
	if cl != nil {
		if env, err := cl.Do("GET", "/api/ServerControlledConfigAPI", nil, nil); err == nil {
			_ = json.Unmarshal(env.Data, &remote)
		}
	}

	ctx.Version, ctx.VersionSource = chooseString(remote.Version, "remote", opts.VersionOverride, "override", opts.ConfigVersion, "config")
	ctx.VersionFamily = ClassifyVersion(ctx.Version)
	if ctx.VersionSource == "remote" && opts.ConfigVersion != "" && opts.ConfigVersion != ctx.Version {
		ctx.Warnings = append(ctx.Warnings, fmt.Sprintf("config version %q differs from remote version %q", opts.ConfigVersion, ctx.Version))
	}

	rawMode := ""
	if len(remote.OperationMode) > 0 {
		rawMode = remote.OperationMode[0]
	}
	modeRaw, modeSource := chooseString(rawMode, "remote", opts.OperationModeOverride, "override", opts.ConfigOperationMode, "config")
	if modeRaw != "" {
		mode, err := NormalizeOperationMode(modeRaw)
		if err != nil {
			return ctx, err
		}
		ctx.OperationMode = mode
		ctx.OperationModeSource = modeSource
	}
	if ctx.OperationModeSource == "remote" && opts.ConfigOperationMode != "" {
		configMode, err := NormalizeOperationMode(opts.ConfigOperationMode)
		if err == nil && configMode != ctx.OperationMode {
			ctx.Warnings = append(ctx.Warnings, fmt.Sprintf("config operation mode %q differs from remote operation mode %q", configMode, ctx.OperationMode))
		}
	}

	if endpoint, ok := EndpointForMode(ctx.OperationMode); ok {
		ctx.Endpoint = endpoint
		ctx.SiteCreateSupported = true
	}
	return ctx, nil
}

func chooseString(a, aSource, b, bSource, c, cSource string) (string, string) {
	if a != "" {
		return a, aSource
	}
	if b != "" {
		return b, bSource
	}
	if c != "" {
		return c, cSource
	}
	return "", ""
}
