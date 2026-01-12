package main

import "slices"

type PartialListFilesResponse []PartialListFileResponse

func (r PartialListFilesResponse) FileByFilename(filename string) (PartialListFileResponse, bool) {
	idx := slices.IndexFunc(r, func(o PartialListFileResponse) bool {
		return o.FileName == filename
	})
	if idx == -1 {
		return PartialListFileResponse{}, false
	}
	return r[idx], true
}

type PartialListFileResponse struct {
	FileName  string   `json:"file_name"`
	Approvals []string `json:"approvals"`
}
