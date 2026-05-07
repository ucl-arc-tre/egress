package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	user1 UserId      = "user-1"
	user2 UserId      = "user-2"
	dest1 Destination = "destination-1"
	dest2 Destination = "destination-2"
)

func TestFileEvents_Approvals(t *testing.T) {
	tests := []struct {
		name     string
		events   FileEvents
		expected FileApprovals
	}{
		{
			name:     "empty",
			events:   FileEvents{},
			expected: FileApprovals{},
		},
		{
			name:     "approve, approve, approve",
			events:   FileEvents{approve(user1, dest1), approve(user1, dest1), approve(user1, dest1)},
			expected: FileApprovals{{UserId: user1, Destination: dest1}},
		},
		{
			name:     "approve, approve, reject",
			events:   FileEvents{approve(user1, dest1), approve(user1, dest1), reject(user1, dest1)},
			expected: FileApprovals{},
		},
		{
			name:     "approve, approve, reject, approve",
			events:   FileEvents{approve(user1, dest1), approve(user1, dest1), reject(user1, dest1), approve(user1, dest1)},
			expected: FileApprovals{{UserId: user1, Destination: dest1}},
		},
		{
			name:     "reject, approve",
			events:   FileEvents{reject(user1, dest1), approve(user1, dest1)},
			expected: FileApprovals{{UserId: user1, Destination: dest1}},
		},
		{
			name:     "approve, download",
			events:   FileEvents{approve(user1, dest1), download(user1, dest1)},
			expected: FileApprovals{{UserId: user1, Destination: dest1}},
		},
		{
			name:     "approve/reject no-duplicates",
			events:   FileEvents{approve(user1, dest1), approve(user2, dest1), reject(user1, dest1), approve(user1, dest2)},
			expected: FileApprovals{{UserId: user2, Destination: dest1}, {UserId: user1, Destination: dest2}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.events.Approvals())
		})
	}
}

func approve(user UserId, dest Destination) Event {
	return Event{
		Action:       EventActionApproval,
		EventDetails: EventDetails{UserId: user, Destination: dest},
	}
}

func reject(user UserId, dest Destination) Event {
	return Event{
		Action:       EventActionRejection,
		EventDetails: EventDetails{UserId: user, Destination: dest},
	}
}

func download(user UserId, dest Destination) Event {
	return Event{
		Action:       EventActionDownload,
		EventDetails: EventDetails{UserId: user, Destination: dest},
	}
}
