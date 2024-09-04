//go:build !worker_runc

package runc

import "embed"

const enabled = false

var assets embed.FS
