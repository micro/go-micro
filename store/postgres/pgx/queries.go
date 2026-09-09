package pgx

import "fmt"

type Queries struct {
	// read
	ListAsc           string
	ListAscLimit      string
	ListDesc          string
	ListDescLimit     string
	ReadOne           string
	ReadManyAsc       string
	ReadManyAscLimit  string
	ReadManyDesc      string
	ReadManyDescLimit string

	// change
	Write         string
	Delete        string
	DeleteExpired string
}

func NewQueries(database, table string) Queries {
	return Queries{
		ListAsc:           fmt.Sprintf(list, database, table) + asc,
		ListAscLimit:      fmt.Sprintf(list, database, table) + asc + limit,
		ListDesc:          fmt.Sprintf(list, database, table) + desc,
		ListDescLimit:     fmt.Sprintf(list, database, table) + desc + limit,
		ReadOne:           fmt.Sprintf(readOne, database, table),
		ReadManyAsc:       fmt.Sprintf(readMany, database, table) + asc,
		ReadManyAscLimit:  fmt.Sprintf(readMany, database, table) + asc + limit,
		ReadManyDesc:      fmt.Sprintf(readMany, database, table) + desc,
		ReadManyDescLimit: fmt.Sprintf(readMany, database, table) + desc + limit,
		Write:             fmt.Sprintf(write, database, table),
		Delete:            fmt.Sprintf(deleteRecord, database, table),
		DeleteExpired:     fmt.Sprintf(deleteExpired, database, table),
	}
}
