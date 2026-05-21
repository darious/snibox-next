package assets

import "embed"

//go:embed all:static
//go:embed seed.json
var FS embed.FS
