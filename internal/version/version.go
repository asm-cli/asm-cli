package version

// These variables are set at build time via ldflags:
//
//	-X 'github.com/6xiaowu9/asm/internal/version.Version=v1.2.3'
//	-X 'github.com/6xiaowu9/asm/internal/version.Commit=abc1234'
//	-X 'github.com/6xiaowu9/asm/internal/version.BuildDate=2025-01-01T00:00:00Z'
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)
