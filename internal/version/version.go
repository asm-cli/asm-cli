package version

// These variables are set at build time via ldflags:
//
//	-X 'github.com/asm-cli/asm-cli/internal/version.Version=v1.2.3'
//	-X 'github.com/asm-cli/asm-cli/internal/version.Commit=abc1234'
//	-X 'github.com/asm-cli/asm-cli/internal/version.BuildDate=2025-01-01T00:00:00Z'
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)
