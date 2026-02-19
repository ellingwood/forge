package embedded

import "embed"

//go:embed all:themes
var DefaultTheme embed.FS
