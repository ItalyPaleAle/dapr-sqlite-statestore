/*
Copyright 2022 The Dapr Authors
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

package component

import (
	"database/sql"
	"fmt"

	"github.com/dapr/components-contrib/state"
)

// Parsed DeleteRequest.
type deleteRequest struct {
	tx        *sql.Tx
	tableName string

	key         string
	concurrency *string
	etag        *string
}

func prepareDeleteRequest(a *sqliteDBAccess, tx *sql.Tx, req *state.DeleteRequest) (*deleteRequest, error) {
	err := state.CheckRequestOptions(req.Options)
	if err != nil {
		return nil, err
	}

	if req.Key == "" {
		return nil, fmt.Errorf("missing key in delete operation")
	}

	if req.Options.Concurrency == state.FirstWrite && (req.ETag == nil || len(*req.ETag) == 0) {
		a.logger.Debugf("when FirstWrite is to be enforced, a value must be provided for the ETag")
		return nil, fmt.Errorf("when FirstWrite is to be enforced, a value must be provided for the ETag")
	}
	return &deleteRequest{
		tx:        tx,
		tableName: a.tableName,

		key:         req.Key,
		concurrency: &req.Options.Concurrency,
		etag:        req.ETag,
	}, nil
}

// Returns if any value deleted, or an execution error.
func (req *deleteRequest) deleteValue() (bool, error) {
	var (
		result sql.Result
		err    error
	)
	if req.etag == nil || *req.etag == "" {
		// Sprintf is required for table name because sql.DB does not substitute parameters for table names.
		stmt := fmt.Sprintf(delValueTpl, req.tableName)
		result, err = req.tx.Exec(stmt, req.key)
	} else {
		// Sprintf is required for table name because sql.DB does not substitute parameters for table names.
		stmt := fmt.Sprintf(delValueWithETagTpl, req.tableName)
		result, err = req.tx.Exec(stmt, req.key, *req.etag)
	}

	if err != nil {
		return false, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows == 1, nil
}
