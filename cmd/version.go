// Copyright Envoy Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v2"
)

var (
	envoyGatewayVersion string
	gatewayAPIVersion   string
)

type Info struct {
	EnvoyGatewayVersion string `json:"envoyGatewayVersion"`
	GatewayAPIVersion   string `json:"gatewayAPIVersion"`
	GolangVersion       string `json:"golangVersion"`
}

func init() {
	bi, ok := debug.ReadBuildInfo()
	if ok {
		for _, dep := range bi.Deps {
			switch dep.Path {
			case "github.com/envoyproxy/gateway":
				envoyGatewayVersion = dep.Version
			case "sigs.k8s.io/gateway-api":
				gatewayAPIVersion = dep.Version
			}
		}
	}
}

// getVersionCommand returns the version cobra command to be executed.
func getVersionCommand() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"versions", "v"},
		Short:   "Show versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return versionPrint(cmd.OutOrStdout(), output)
		},
	}

	cmd.PersistentFlags().StringVarP(&output, "output", "o", "", "One of 'yaml' or 'json'")

	return cmd
}

func Get() Info {
	return Info{
		EnvoyGatewayVersion: envoyGatewayVersion,
		GatewayAPIVersion:   gatewayAPIVersion,
		GolangVersion:       runtime.Version(),
	}
}

// versionPrint shows the versions of the Envoy Gateway, Gateway API.
func versionPrint(w io.Writer, format string) error {
	v := Get()
	switch format {
	case "json":
		if marshaled, err := json.MarshalIndent(v, "", "  "); err == nil {
			_, _ = fmt.Fprintln(w, string(marshaled))
		}
	case "yaml":
		if marshaled, err := yaml.Marshal(v); err == nil {
			_, _ = fmt.Fprintln(w, string(marshaled))
		}
	default:
		_, _ = fmt.Fprintf(w, "ENVOY_GATEWAY_VERSION: %s\n", v.EnvoyGatewayVersion)
		_, _ = fmt.Fprintf(w, "GATEWAYAPI_VERSION: %s\n", v.GatewayAPIVersion)
		_, _ = fmt.Fprintf(w, "GOLANG_VERSION: %s\n", v.GolangVersion)
	}
	return nil
}
