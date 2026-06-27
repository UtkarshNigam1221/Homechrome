// Package stacks builds the AWS CDK stacks for handloom-admin.
package stacks

import (
	"fmt"
)

// OtelCollectorLayerArn returns the ARN of the community-published OpenTelemetry
// Collector Lambda extension layer for the given region + architecture.
// Source: https://github.com/open-telemetry/opentelemetry-lambda/releases
//
// Account ID 184161586896 is the OpenTelemetry community publisher.
// Latest release: layer-collector/0.22.0 (May 8, 2025), contains OTel Collector v1.57.0/v0.151.0.
// TODO: bump version + revision when a newer layer-collector release is published.
func OtelCollectorLayerArn(region, arch string) string {
	// CHOSEN_VERSION — update when bumping
	const version = "0_22_0"
	const revision = "1"
	return fmt.Sprintf(
		"arn:aws:lambda:%s:184161586896:layer:opentelemetry-collector-%s-%s:%s",
		region, arch, version, revision,
	)
}
