package openapi

import "github.com/ucl-arc-tre/egress/internal/types"

//go:generate go tool oapi-codegen -generate types,spec,gin -package openapi -o main.gen.go ../../api/api.yaml

func MakeFileMetadata(objectMeta types.ObjectMeta, approvals types.FileApprovals) FileMetadata {
	fileMetadata := FileMetadata{
		FileName: objectMeta.Name,
		Id:       string(objectMeta.Id),
		Size:     objectMeta.NumBytes,
	}
	for _, approval := range approvals {
		fileMetadata.Approvals = append(fileMetadata.Approvals, string(approval))
	}
	return fileMetadata
}
