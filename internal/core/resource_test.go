package core

import "testing"

func TestResourceStatusTransitions(t *testing.T) {
	valid := []struct {
		from ResourceStatus
		to   ResourceStatus
	}{
		{ResourceStatusRequested, ResourceStatusProvisioning},
		{ResourceStatusRequested, ResourceStatusFailed},
		{ResourceStatusProvisioning, ResourceStatusReady},
		{ResourceStatusProvisioning, ResourceStatusFailed},
		{ResourceStatusReady, ResourceStatusUpdating},
		{ResourceStatusReady, ResourceStatusDeleting},
		{ResourceStatusUpdating, ResourceStatusReady},
		{ResourceStatusUpdating, ResourceStatusFailed},
		{ResourceStatusDeleting, ResourceStatusDeleted},
		{ResourceStatusDeleting, ResourceStatusFailed},
		{ResourceStatusFailed, ResourceStatusRequested},
		{ResourceStatusFailed, ResourceStatusDeleting},
	}
	for _, tt := range valid {
		if !tt.from.CanTransition(tt.to) {
			t.Errorf("expected %s → %s to be valid", tt.from, tt.to)
		}
	}

	invalid := []struct {
		from ResourceStatus
		to   ResourceStatus
	}{
		{ResourceStatusRequested, ResourceStatusReady},
		{ResourceStatusRequested, ResourceStatusDeleting},
		{ResourceStatusProvisioning, ResourceStatusDeleting},
		{ResourceStatusReady, ResourceStatusRequested},
		{ResourceStatusReady, ResourceStatusProvisioning},
		{ResourceStatusDeleted, ResourceStatusReady},
		{ResourceStatusDeleted, ResourceStatusRequested},
	}
	for _, tt := range invalid {
		if tt.from.CanTransition(tt.to) {
			t.Errorf("expected %s → %s to be invalid", tt.from, tt.to)
		}
	}
}
