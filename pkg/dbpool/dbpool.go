/*
Copyright 2018 Ridecell, Inc..

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dbpool

import (
	"database/sql"
	"fmt"
	"sync"
)

var Dbs sync.Map

func Open(driverName, dataSourceName string) (*sql.DB, error) {
	key := fmt.Sprintf("%s %s", driverName, dataSourceName)
	// First pass, check if the key is available at all.
	mapVal, ok := Dbs.Load(key)
	if ok {
		// Connection already present.
		return mapVal.(*sql.DB), nil
	}
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		// Welp, we tried.
		return nil, err
	}
	mapVal, loaded := Dbs.LoadOrStore(key, db)
	if loaded {
		// Race, someone else got there first. Clean up and bail.
		db.Close()
		return mapVal.(*sql.DB), nil
	} else {
		// Success, stored, we're done.
		return db, nil
	}
}
