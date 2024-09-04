//go:build worker_runc

package runc

import "embed"

const enabled = true

//go:embed assets/*
var assets embed.FS
