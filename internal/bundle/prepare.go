package bundle

import (
	"fmt"
)

// ValidatePlatforms checks that each platform is one of linux64/osx64/win64.
func ValidatePlatforms(platforms []string) error {
	for _, p := range platforms {
		switch p {
		case "linux64", "osx64", "win64":
		default:
			return fmt.Errorf("unknown platform %q; must be one of: linux64, osx64, win64", p)
		}
	}
	return nil
}
