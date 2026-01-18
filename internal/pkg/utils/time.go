package utils

import (
	"fmt"
	"os"
	"time"
)

var Beijing *time.Location

// SetupTimeLocations initializes timezone locations.
func SetupTimeLocations() {
	var err error
	if Beijing, err = time.LoadLocation("Asia/Shanghai"); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: tzdata missing, cannot load Asia/Shanghai: %v\n", err)
		os.Exit(1)
	}
}
