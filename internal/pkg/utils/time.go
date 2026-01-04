package utils

import (
	"time"
)

// time locations
var (
	Beijing *time.Location
)

// SetupTimeLocations setups time locations
func SetupTimeLocations() (err error) {
	if Beijing, err = time.LoadLocation("Asia/Shanghai"); err != nil {
		return
	}
	return nil
}
