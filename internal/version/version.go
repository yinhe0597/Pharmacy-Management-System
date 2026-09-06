// Package version 承载构建版本信息，由构建期 ldflags 注入（见 Makefile/Dockerfile/CI）。
package version

// 以下变量在构建时通过 -ldflags "-X yaofang/internal/version.Version=..." 覆盖。
var (
	// Version 语义化版本号；未注入时为 dev。
	Version = "dev"
	// Commit 构建对应的 git commit（短哈希）。
	Commit = "unknown"
	// BuildTime 构建时间（RFC3339）。
	BuildTime = "unknown"
)
