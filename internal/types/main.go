package types

type ProjectId string

type UserId string

type FileId string

type FileApprovals []UserId

type ProjectApprovals map[FileId]FileApprovals
