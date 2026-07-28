//go:build !embed_frontend

package main

import "io/fs"

func getFrontendFS() (fs.FS, error) {
	return nil, nil
}
