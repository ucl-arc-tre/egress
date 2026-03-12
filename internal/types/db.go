package types

type DBProvider string

const (
	DBProviderInMemory = DBProvider("inmemory")
	DBProviderRqlite   = DBProvider("rqlite")
)
