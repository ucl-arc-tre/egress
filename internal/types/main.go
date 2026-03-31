package types

// Unique identifier of a project.
// A project is a collection of people/data that share
// the same set of egress approvals
type ProjectId string

// Unique identifier of a user. May or may not be unique
// across projects
type UserId string

// Destination for which a file can be egressed
type Destination string

// An egress approval, recording the approving user
// and the destination for which it is approved
type Approval struct {
	UserId      UserId
	Destination Destination
}

// List of approvals granted for a file
type FileApprovals []Approval

// Get approvals for the given destination
func (f FileApprovals) ForDestination(destination Destination) FileApprovals {
	filtered := FileApprovals{}
	for _, approval := range f {
		if approval.Destination == destination {
			filtered = append(filtered, approval)
		}
	}
	return filtered
}

// Map of files to a list of approvals granted for the file
type ProjectApprovals map[FileId]FileApprovals

// Get the file approvals for a particular file. If it doesn't
// exist then get returns an empty set of file approvals
func (p ProjectApprovals) FileApprovals(fileId FileId) FileApprovals {
	approvals, exists := p[fileId]
	if !exists {
		return FileApprovals{}
	}
	return approvals
}
