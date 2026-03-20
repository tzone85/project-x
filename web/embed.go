// Package web embeds static dashboard assets (HTML, JS, CSS) so they
// ship inside the compiled binary. No external files needed at runtime.
package web

import "embed"

//go:embed index.html app.js style.css
var Assets embed.FS
