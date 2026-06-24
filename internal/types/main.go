package types

import (
	"sort"
	"time"
)

// Unique identifier of a project.
// A project is a collection of people/data that share
// the same set of egress approvals
type ProjectId string

// Unique identifier of a user. May or may not be unique
// across projects
type UserId string

// Destination for which a file can be egressed
type Destination string

// Describes an egress related event tracked at file level
// An egress event is either an approval, a rejection or a download
type Event struct {
	Time   time.Time
	Action EventAction
	EventDetails
}

// Describes the details of an event
type EventDetails struct {
	UserId      UserId
	Destination Destination
	Comment     string
}

// The specific action of an event
type EventAction string

// Supported event actions
const (
	EventActionApproval  EventAction = "Approval"
	EventActionDownload  EventAction = "Download"
	EventActionRejection EventAction = "Rejection"
)

// An egress file approval, recording the approving user
// and the destination for which it is approved
// An approval is a type of an egress event
type Approval EventDetails

// List of egress events associated with a file
type FileEvents []Event

// Get approvals of a file
// Multiple approvals with the same {UserId, Destination} are de-duplicated
// A rejection that comes after an approval cancels that approval
// Events are sorted chronologically by Time before processing
func (fe FileEvents) Approvals() FileApprovals {
	type approvalKey struct {
		userId      UserId
		destination Destination
	}
	latest := map[approvalKey]Approval{}
	approved := map[approvalKey]bool{}
	order := []approvalKey{}

	sorted := make(FileEvents, len(fe))
	copy(sorted, fe)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Time.Before(sorted[j].Time)
	})

	for _, e := range sorted {
		if e.Action != EventActionApproval && e.Action != EventActionRejection {
			continue
		}
		key := approvalKey{userId: e.UserId, destination: e.Destination}
		if _, seen := latest[key]; !seen {
			order = append(order, key)
		}
		// Keep the most recent details so the returned approval reflects the
		// latest approval's comment, not the first one for this key
		latest[key] = Approval(e.EventDetails)
		approved[key] = e.Action == EventActionApproval
	}
	// Return filtered approvals in the same order as input
	approvals := FileApprovals{}
	for _, key := range order {
		if approved[key] {
			approvals = append(approvals, latest[key])
		}
	}
	return approvals
}

// List of approvals granted for a file
type FileApprovals []Approval

// Get approvals for the given destination
func (fa FileApprovals) ForDestination(destination Destination) FileApprovals {
	filtered := FileApprovals{}
	for _, approval := range fa {
		if approval.Destination == destination {
			filtered = append(filtered, approval)
		}
	}
	return filtered
}

// Map of files to a list of events associated with the file
type ProjectEvents map[FileId]FileEvents

// Return approvals granted for all the files in the project
func (pe ProjectEvents) ProjectApprovals() ProjectApprovals {
	approvals := ProjectApprovals{}
	for fileId, events := range pe {
		approvals[fileId] = events.Approvals()
	}
	return approvals
}

// Map of files to a list of approvals granted for the file
type ProjectApprovals map[FileId]FileApprovals

// Get the file approvals for a particular file. If it doesn't
// exist then return an empty set of file approvals
func (pa ProjectApprovals) FileApprovals(fileId FileId) FileApprovals {
	approvals, exists := pa[fileId]
	if !exists {
		return FileApprovals{}
	}
	return approvals
}
