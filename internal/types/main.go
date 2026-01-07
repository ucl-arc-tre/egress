package types

// Unique identifier of a project.
// A project is a collection of people/data aht share
// the same set of egress approvals
type ProjectId string

// Unique identifier of a user. May or may not be unique
// accross projects
type UserId string

// Unique file identifier. e.g. a SHA512 checksum
type FileId string

// List of users who have approved a file
type FileApprovals []UserId

// Map of files to a list of users who approved the file
type ProjectApprovals map[FileId]FileApprovals

// Get the file approvals for particular file. If if doesn't
// exist then get returns an empty set of file approvals
func (p ProjectApprovals) FileApprovals(fileId FileId) FileApprovals {
	approvals, exists := p[fileId]
	if !exists {
		return FileApprovals{}
	}
	return approvals
}
