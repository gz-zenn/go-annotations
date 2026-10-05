package embed

import "embed"

// Embedding a single file as a string: the shape to use for config defaults,
// SQL migrations, prompt templates or anything else that is text.
//
//go:embed config/default.yaml
var defaultConfig string

// Embedding as []byte instead: handy for binary assets, where a string copy
// would be a waste.
//
//go:embed assets/logo.png
var logoBytes []byte

// Several patterns on one directive: directories and file types can be
// combined, and everything lands in a single FS.
//
//go:embed templates/*.html static/css/*.css static/js/*.js
var webAssets embed.FS

// DefaultConfig returns the embedded configuration file as a string.
func DefaultConfig() string { return defaultConfig }

// Logo returns the embedded PNG as bytes.
func Logo() []byte { return logoBytes }

// WebAssets returns the multi-pattern FS.
func WebAssets() embed.FS { return webAssets }
