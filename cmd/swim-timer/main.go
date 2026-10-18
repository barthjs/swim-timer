package main

import (
	"os"
	// Embed the IANA time zone database into the binary so time zones resolve
	// on minimal base images like alpine that ship without the tzdata package.
	_ "time/tzdata"

	"github.com/barthjs/swim-timer/internal/cli"
)

// Version is stamped into the binary at build time.
var Version = "dev"

// @license.name				MIT
// @BasePath					/api
//
// @securityDefinitions.apikey	SessionAuth
// @in							cookie
// @name						session
//
//nolint:godot // The general API info for swag ends with an annotation.
func main() {
	os.Exit(cli.Execute(Version))
}
