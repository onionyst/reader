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
	Beijing, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return err
	}

	return nil
}
