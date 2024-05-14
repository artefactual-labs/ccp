package controller

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// uuidFromPath returns the UUID if it's the suffix of the path.
func uuidFromPath(path string) uuid.UUID {
	path = strings.TrimRight(path, string(os.PathSeparator))
	if len(path) < 36 {
		return uuid.Nil
	}
	id, err := uuid.Parse(path[len(path)-36:])
	if err != nil {
		return uuid.Nil
	}
	return id
}

// joinPath is like filepath.Join but appends the ending separator when the last
// element provided is an empty string or ends with a slash.
func joinPath(elem ...string) string {
	const sep = string(os.PathSeparator)
	ln := len(elem)
	if ln == 0 {
		return ""
	}
	ret := filepath.Join(elem...)
	last := elem[ln-1]
	if last == "" || strings.HasSuffix(last, sep) {
		ret += sep
	}

	return ret
}

// locationPath extracts the location identifier and path from a locationPath.
// It returs the nil value of uuid.UUID (uuid.Nil) when the identifier is not
// part of the path.
func locationPath(locationPath string) (id uuid.UUID, path string) {
	before, after, found := strings.Cut(locationPath, ":")

	if found {
		id, _ = uuid.Parse(before)
		path = after
	} else {
		path = before
	}

	return id, path
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
