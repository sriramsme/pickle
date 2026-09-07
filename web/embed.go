package web

import "embed"

// Dist contains the production frontend built by Vite.
//
//go:embed all:dist
var Dist embed.FS
