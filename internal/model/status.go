package model

// StatusState is the 4-state semaphore shown for each tool row.
type StatusState int

const (
	StatusUpToDate    StatusState = iota // green: installed and current
	StatusUpdateAvail                    // yellow: installed, newer version known
	StatusNotInstalled                   // red: not found on this machine
	StatusNoUpdater                      // white: installed but no native update path known
)

func (s StatusState) String() string {
	switch s {
	case StatusUpToDate:
		return "up-to-date"
	case StatusUpdateAvail:
		return "update-available"
	case StatusNotInstalled:
		return "not-installed"
	case StatusNoUpdater:
		return "no-updater"
	default:
		return "unknown"
	}
}
