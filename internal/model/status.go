package model

// StatusState is the 4-state semaphore shown for each tool row.
type StatusState int

const (
	StatusUpToDate      StatusState = iota // green: installed and current
	StatusUpdateAvail                      // yellow: installed, a newer version is known
	StatusNotInstalled                     // red: not found on this machine
	StatusLatestUnknown                    // white: installed but the latest version could not be verified
)

func (s StatusState) String() string {
	switch s {
	case StatusUpToDate:
		return "up-to-date"
	case StatusUpdateAvail:
		return "update-available"
	case StatusNotInstalled:
		return "not-installed"
	case StatusLatestUnknown:
		return "latest-unknown"
	default:
		return "unknown"
	}
}
