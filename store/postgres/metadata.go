// Copyright 2020 Asim Aslam
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Original source: github.com/micro/go-plugins/v3/store/cockroach/metadata.go

package postgres

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// https://github.com/upper/db/blob/master/postgresql/custom_types.go#L43
type Metadata map[string]interface{}

// Scan satisfies the sql.Scanner interface.
func (m *Metadata) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("Type assertion .([]byte) failed.")
	}

	var i interface{}
	err := json.Unmarshal(source, &i)
	if err != nil {
		return err
	}

	*m, ok = i.(map[string]interface{})
	if !ok {
		return errors.New("Type assertion .(map[string]interface{}) failed.")
	}

	return nil
}

// Value satisfies the driver.Valuer interface.
func (m Metadata) Value() (driver.Value, error) {
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
