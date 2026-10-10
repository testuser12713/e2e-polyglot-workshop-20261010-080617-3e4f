// Package domain holds the workshop's core value types that every layer of the
// API shares.
package domain

// Status is the lifecycle state of a workshop order.
type Status string

const (
	StatusRequested  Status = "angefragt"
	StatusConfirmed  Status = "bestaetigt"
	StatusInProgress Status = "in_arbeit"
	StatusDone       Status = "fertig"
	StatusPickedUp   Status = "abgeholt"
)

// AllStatuses lists every valid status in lifecycle order.
var AllStatuses = []Status{
	StatusRequested,
	StatusConfirmed,
	StatusInProgress,
	StatusDone,
	StatusPickedUp,
}

// IsValid reports whether s is one of the five known statuses.
func IsValid(s Status) bool {
	for _, candidate := range AllStatuses {
		if s == candidate {
			return true
		}
	}
	return false
}

// AllowedNext returns the single status that s may transition to. The second
// result is false for the terminal status and for any unknown status.
func (s Status) AllowedNext() (Status, bool) {
	for i, candidate := range AllStatuses {
		if s == candidate && i+1 < len(AllStatuses) {
			return AllStatuses[i+1], true
		}
	}
	return "", false
}
