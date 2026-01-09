package openapi

import "github.com/ucl-arc-tre/egress/internal/types"

//go:generate go tool oapi-codegen -generate types,spec,gin -package openapi -o main.gen.go ../../api/api.yaml

func MakeFileMetadata(metadata types.FileMetadata, approvals types.FileApprovals) FileMetadata {
	fileMetadata := FileMetadata{
		FileName:  metadata.Name,
		Id:        string(metadata.Id),
		Size:      int(metadata.Size),
		Approvals: []string{},
	}
	for _, approval := range approvals {
		fileMetadata.Approvals = append(fileMetadata.Approvals, string(approval))
	}
	return fileMetadata
}
