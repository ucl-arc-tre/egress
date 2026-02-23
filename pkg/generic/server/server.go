// Package server provides a minimal implementation of a generic storage server
package server

//go:generate go tool oapi-codegen -generate gin -package server -o server.gen.go ../../../api/storage.yaml
//go:generate go tool oapi-codegen -generate spec -package server -o spec.gen.go ../../../api/storage.yaml
//go:generate go tool oapi-codegen -generate types -package server -o types.gen.go ../../../api/storage.yaml
