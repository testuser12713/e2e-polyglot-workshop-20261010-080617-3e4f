// Package status implements the workshop order status lifecycle on top of the
// shared domain.Status value type. The lifecycle is a strict single step:
// angefragt -> bestaetigt -> in_arbeit -> fertig -> abgeholt. No jump forward
// and no step backwards is ever allowed.
package status

import (
	"errors"
	"fmt"

	"workshop/internal/domain"
)

// ErrInvalidTransition is returned for any transition the lifecycle forbids:
// a jump forward, a step backwards, or an unknown target status.
var ErrInvalidTransition = errors.New("status: invalid transition")

// CanTransition reports whether from may advance to to in exactly one legal
// step. It is built on domain.Status.AllowedNext, so there is a single source
// of truth for the lifecycle order.
func CanTransition(from, to domain.Status) bool {
	if !domain.IsValid(from) || !domain.IsValid(to) {
		return false
	}
	next, ok := from.AllowedNext()
	return ok && next == to
}

// Transition validates the single step from -> to and returns the resulting
// status. On an illegal transition it returns an error wrapping
// ErrInvalidTransition and leaves the order's status untouched.
func Transition(from, to domain.Status) (domain.Status, error) {
	if !domain.IsValid(to) {
		return from, fmt.Errorf("%w: unknown status %q", ErrInvalidTransition, to)
	}
	if !CanTransition(from, to) {
		return from, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
	}
	return to, nil
}
