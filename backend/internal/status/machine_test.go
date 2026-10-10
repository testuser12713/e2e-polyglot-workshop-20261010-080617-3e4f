package status

import (
	"errors"
	"testing"

	"workshop/internal/domain"
)

func TestCanTransitionAllowsOnlySingleForwardStep(t *testing.T) {
	allowed := map[[2]domain.Status]bool{
		{domain.StatusRequested, domain.StatusConfirmed}:  true,
		{domain.StatusConfirmed, domain.StatusInProgress}: true,
		{domain.StatusInProgress, domain.StatusDone}:      true,
		{domain.StatusDone, domain.StatusPickedUp}:        true,
	}

	for _, from := range domain.AllStatuses {
		for _, to := range domain.AllStatuses {
			want := allowed[[2]domain.Status{from, to}]
			if got := CanTransition(from, to); got != want {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", from, to, got, want)
			}
		}
	}
}

func TestCanTransitionRejectsUnknownStatus(t *testing.T) {
	if CanTransition(domain.StatusRequested, domain.Status("nonsense")) {
		t.Error("CanTransition accepted an unknown target status")
	}
	if CanTransition(domain.Status("nonsense"), domain.StatusConfirmed) {
		t.Error("CanTransition accepted an unknown source status")
	}
}

func TestTransitionReturnsTargetOnLegalStep(t *testing.T) {
	got, err := Transition(domain.StatusRequested, domain.StatusConfirmed)
	if err != nil {
		t.Fatalf("Transition(angefragt, bestaetigt) returned error: %v", err)
	}
	if got != domain.StatusConfirmed {
		t.Errorf("Transition result = %q, want %q", got, domain.StatusConfirmed)
	}
}

func TestTransitionRejectsJumpAndBackwards(t *testing.T) {
	cases := []struct {
		name string
		from domain.Status
		to   domain.Status
	}{
		{"jump", domain.StatusRequested, domain.StatusInProgress},
		{"jump to end", domain.StatusRequested, domain.StatusPickedUp},
		{"backwards", domain.StatusConfirmed, domain.StatusRequested},
		{"same status", domain.StatusConfirmed, domain.StatusConfirmed},
		{"terminal", domain.StatusPickedUp, domain.StatusDone},
		{"unknown target", domain.StatusRequested, domain.Status("nonsense")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Transition(tc.from, tc.to)
			if !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf("Transition(%s, %s) error = %v, want ErrInvalidTransition", tc.from, tc.to, err)
			}
			if got != tc.from {
				t.Errorf("Transition(%s, %s) returned %q, want the unchanged %q", tc.from, tc.to, got, tc.from)
			}
		})
	}
}
