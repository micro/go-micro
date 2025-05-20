package pgx

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type Metadata map[string]interface{}

// Scan satisfies the sql.Scanner interface.
func (m *Metadata) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("type assertion .([]byte) failed")
	}

	var i interface{}
	err := json.Unmarshal(source, &i)
	if err != nil {
		return err
	}

	*m, ok = i.(map[string]interface{})
	if !ok {
		return errors.New("type assertion .(map[string]interface{}) failed")
	}

	return nil
}

// Value satisfies the driver.Valuer interface.
func (m *Metadata) Value() (driver.Value, error) {
	j, err := json.Marshal(m)
	return j, err
}

func toMetadata(m *Metadata) map[string]interface{} {
	md := make(map[string]interface{})
	for k, v := range *m {
		md[k] = v
	}
	return md
}
