// Package web embeds the static single-page UI.
package web

import "embed"

// Files holds the UI assets served at /.
//
//go:embed index.html styles.css app.js
var Files embed.FS
