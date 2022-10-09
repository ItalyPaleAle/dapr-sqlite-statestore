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
	"context"

	"github.com/dapr/components-contrib/state"
	"github.com/dapr/kit/logger"
)

// SQLite Database state store.
type SQLiteStore struct {
	features []state.Feature
	logger   logger.Logger
	dbaccess DBAccess
}

// NewSQLiteStateStore creates a new instance of the SQLite state store.
func NewSQLiteStateStore(logger logger.Logger) state.Store {
	dba := newSqliteDBAccess(logger)

	return newSQLiteStateStore(logger, dba)
}

// newSQLiteStateStore creates a newSQLiteStateStore instance of an Sqlite state store.
// This unexported constructor allows injecting a dbAccess instance for unit testing.
func newSQLiteStateStore(logger logger.Logger, dba DBAccess) *SQLiteStore {
	return &SQLiteStore{
		logger:   logger,
		dbaccess: dba,
	}
}

// Init initializes the Sql server state store.
func (s *SQLiteStore) Init(metadata state.Metadata) error {
	return s.dbaccess.Init(metadata)
}

func (s *SQLiteStore) Ping() error {
	return s.dbaccess.Ping(context.TODO())
}

// Features returns the features available in this state store.
func (s *SQLiteStore) Features() []state.Feature {
	return []state.Feature{
		state.FeatureETag,
		state.FeatureTransactional,
	}
}

// Delete removes an entity from the store.
func (s *SQLiteStore) Delete(req *state.DeleteRequest) error {
	return s.dbaccess.Delete(context.TODO(), req)
}

// BulkDelete removes multiple entries from the store.
func (s *SQLiteStore) BulkDelete(req []state.DeleteRequest) error {
	var ops = make([]state.TransactionalStateOperation, len(req))
	for _, r := range req {
		ops = append(ops, state.TransactionalStateOperation{
			Operation: state.Delete,
			Request:   r,
		})
	}
	return s.dbaccess.ExecuteMulti(context.TODO(), ops)
}

// Get returns an entity from store.
func (s *SQLiteStore) Get(req *state.GetRequest) (*state.GetResponse, error) {
	return s.dbaccess.Get(context.TODO(), req)
}

// BulkGet performs a bulks get operations.
func (s *SQLiteStore) BulkGet(req []state.GetRequest) (bool, []state.BulkGetResponse, error) {
	// TODO: replace with ExecuteMulti for performance.
	return false, nil, nil
}

// Set adds/updates an entity on store.
func (s *SQLiteStore) Set(req *state.SetRequest) error {
	return s.dbaccess.Set(context.TODO(), req)
}

// BulkSet adds/updates multiple entities on store.
func (s *SQLiteStore) BulkSet(req []state.SetRequest) error {
	var ops = make([]state.TransactionalStateOperation, len(req))
	for _, r := range req {
		ops = append(ops, state.TransactionalStateOperation{
			Operation: state.Upsert,
			Request:   r,
		})
	}
	return s.dbaccess.ExecuteMulti(context.TODO(), ops)
}

// Multi handles multiple transactions. Implements TransactionalStore.
func (s *SQLiteStore) Multi(request *state.TransactionalStateRequest) error {
	return s.dbaccess.ExecuteMulti(context.TODO(), request.Operations)
}

// Close implements io.Closer.
func (s *SQLiteStore) Close() error {
	if s.dbaccess != nil {
		return s.dbaccess.Close()
	}

	return nil
}
