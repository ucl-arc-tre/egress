package types

import (
	"testing"
	"time"

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
		{
			// Regression for #78: the latest approval's comment must win
			name: "approve, reject, approve keeps latest comment",
			events: FileEvents{
				approveWithComment(user1, dest1, "first"),
				rejectWithComment(user1, dest1, "second"),
				approveWithComment(user1, dest1, "third"),
			},
			expected: FileApprovals{{UserId: user1, Destination: dest1, Comment: "third"}},
		},
		{
			// Regression for #78: consecutive approvals keep the latest comment
			name: "approve, approve keeps latest comment",
			events: FileEvents{
				approveWithComment(user1, dest1, "first"),
				approveWithComment(user1, dest1, "second"),
			},
			expected: FileApprovals{{UserId: user1, Destination: dest1, Comment: "second"}},
		},
		{
			// Distinct comments per key, preserving first-appearance ordering
			name: "multiple keys keep their own latest comments in order",
			events: FileEvents{
				approveWithComment(user1, dest1, "u1d1-a"),
				approveWithComment(user2, dest1, "u2d1-a"),
				reject(user1, dest1),
				approveWithComment(user1, dest1, "u1d1-b"),
				approveWithComment(user2, dest1, "u2d1-b"),
			},
			expected: FileApprovals{
				{UserId: user1, Destination: dest1, Comment: "u1d1-b"},
				{UserId: user2, Destination: dest1, Comment: "u2d1-b"},
			},
		},
		{
			name: "events are chronologically ordered",
			events: FileEvents{
				// Approval happened after the rejection, but comes first in the slice
				approveAt(user1, dest1, time.Unix(3, 0)),
				rejectAt(user1, dest1, time.Unix(1, 0)),
			},
			expected: FileApprovals{{UserId: user1, Destination: dest1}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.events.Approvals())
		})
	}
}

func approve(user UserId, dest Destination) Event {
	return approveWithComment(user, dest, "")
}

func reject(user UserId, dest Destination) Event {
	return rejectWithComment(user, dest, "")
}

func download(user UserId, dest Destination) Event {
	return Event{
		Action:       EventActionDownload,
		EventDetails: EventDetails{UserId: user, Destination: dest},
	}
}

func approveWithComment(user UserId, dest Destination, comment string) Event {
	return Event{
		Action:       EventActionApproval,
		EventDetails: EventDetails{UserId: user, Destination: dest, Comment: comment},
	}
}

func rejectWithComment(user UserId, dest Destination, comment string) Event {
	return Event{
		Action:       EventActionRejection,
		EventDetails: EventDetails{UserId: user, Destination: dest, Comment: comment},
	}
}

func approveAt(user UserId, dest Destination, t time.Time) Event {
	return Event{
		Action:       EventActionApproval,
		EventDetails: EventDetails{UserId: user, Destination: dest},
		Time:         t,
	}
}

func rejectAt(user UserId, dest Destination, t time.Time) Event {
	return Event{
		Action:       EventActionRejection,
		EventDetails: EventDetails{UserId: user, Destination: dest},
		Time:         t,
	}
}
