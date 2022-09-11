// Author: https://belief-driven-design.com/build-time-variables-in-go-51439b26ef9/

package version

import (
	"fmt"
)

var (
	Version        = "dev"
	CommitHash     = "n/a"
	BuildTimestamp = "n/a"
)

func BuildVersion() string {
	return fmt.Sprintf("%s-%s (%s)", Version, CommitHash, BuildTimestamp)
}
