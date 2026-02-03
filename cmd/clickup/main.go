package main

import (
	"os"

	"github.com/triptechtravel/clickup-cli/internal/app"
)

func main() {
	code := app.Run()
	os.Exit(code)
}
