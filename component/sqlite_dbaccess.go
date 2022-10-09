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
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/dapr/components-contrib/state"
	"github.com/dapr/kit/logger"

	// Blank import for the underlying SQLite Driver.
	_ "github.com/mattn/go-sqlite3"
)

// DBAccess is a private interface which enables unit testing of SQLite.
type DBAccess interface {
	Init(metadata state.Metadata) error
	Ping(ctx context.Context) error
	Set(ctx context.Context, req *state.SetRequest) error
	Get(ctx context.Context, req *state.GetRequest) (*state.GetResponse, error)
	Delete(ctx context.Context, req *state.DeleteRequest) error
	ExecuteMulti(ctx context.Context, reqs []state.TransactionalStateOperation) error
	Close() error
}

// sqliteDBAccess implements DBAccess.
type sqliteDBAccess struct {
	logger           logger.Logger
	metadata         state.Metadata
	connectionString string
	tableName        string
	db               *sql.DB
	cleanupInterval  *time.Duration
	ctx              context.Context
	cancel           context.CancelFunc

	// Lock only on public write API. Any public API's implementation should not call other public write APIs.
	lock *sync.RWMutex
}

// newSqliteDBAccess creates a new instance of sqliteDbAccess.
func newSqliteDBAccess(logger logger.Logger) *sqliteDBAccess {
	return &sqliteDBAccess{
		logger: logger,
		lock:   &sync.RWMutex{},
	}
}

// Init sets up SQLite Database connection and ensures that the state table
// exists.
func (a *sqliteDBAccess) Init(metadata state.Metadata) error {
	a.metadata = metadata

	tableName, ok := metadata.Properties[tableNameKey]
	if !ok || tableName == "" {
		tableName = defaultTableName
	} else if !validIdentifier(tableName) {
		return fmt.Errorf(errInvalidIdentifier, tableName)
	}
	a.tableName = tableName

	cleanupInterval, err := a.parseCleanupInterval(metadata)
	if err != nil {
		return err
	}
	a.cleanupInterval = cleanupInterval

	if val, ok := metadata.Properties[connectionStringKey]; ok && val != "" {
		a.connectionString = val
	} else {
		a.logger.Error("Missing SQLite connection string")

		return errors.New(errMissingConnectionString)
	}

	db, err := sql.Open("sqlite3", a.connectionString)
	if err != nil {
		a.logger.Error(err)
		return err
	}

	a.db = db
	a.ctx, a.cancel = context.WithCancel(context.Background())

	if pingErr := a.Ping(a.ctx); pingErr != nil {
		return pingErr
	}

	err = a.ensureStateTable(a.ctx, tableName)
	if err != nil {
		return err
	}

	a.scheduleCleanupExpiredData()

	return nil
}

func (a *sqliteDBAccess) Ping(parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(parentCtx, operationTimeout)
	err := a.db.PingContext(ctx)
	cancel()
	return err
}

func (a *sqliteDBAccess) Get(parentCtx context.Context, req *state.GetRequest) (*state.GetResponse, error) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	if req.Key == "" {
		return nil, errors.New("missing key in get operation")
	}
	var (
		value    string
		isBinary bool
		etag     string
	)

	// Sprintf is required for table name because sql.DB does not substitute parameters for table names.
	stmt := fmt.Sprintf(getValueTpl, a.tableName)
	ctx, cancel := context.WithTimeout(parentCtx, operationTimeout)
	err := a.db.QueryRowContext(ctx, stmt, req.Key).Scan(&value, &isBinary, &etag)
	cancel()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &state.GetResponse{
				Metadata: req.Metadata,
			}, nil
		}
		return nil, err
	}
	if isBinary {
		var s string
		var data []byte
		if err = json.Unmarshal([]byte(value), &s); err != nil {
			return nil, err
		}
		if data, err = base64.StdEncoding.DecodeString(s); err != nil {
			return nil, err
		}
		return &state.GetResponse{
			Data:     data,
			ETag:     &etag,
			Metadata: req.Metadata,
		}, nil
	}
	return &state.GetResponse{
		Data:     []byte(value),
		ETag:     &etag,
		Metadata: req.Metadata,
	}, nil
}

func (a *sqliteDBAccess) Set(parentCtx context.Context, req *state.SetRequest) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	ctx, cancel := context.WithTimeout(parentCtx, operationTimeout)
	defer cancel()

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = state.SetWithOptions(
		func(req *state.SetRequest) error {
			return a.setValue(tx, req)
		},
		req,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (a *sqliteDBAccess) Delete(parentCtx context.Context, req *state.DeleteRequest) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	ctx, cancel := context.WithTimeout(parentCtx, operationTimeout)
	defer cancel()

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = state.DeleteWithOptions(
		func(req *state.DeleteRequest) error {
			return a.deleteValue(tx, req)
		},
		req,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (a *sqliteDBAccess) ExecuteMulti(parentCtx context.Context, reqs []state.TransactionalStateOperation) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	ctx, cancel := context.WithTimeout(parentCtx, operationTimeout)
	defer cancel()

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, req := range reqs {
		switch req.Operation {
		case state.Upsert:
			if setReq, ok := req.Request.(state.SetRequest); ok {
				err = a.setValue(tx, &setReq)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("expecting set request")
			}
		case state.Delete:
			if delReq, ok := req.Request.(state.DeleteRequest); ok {
				err = a.deleteValue(tx, &delReq)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("expecting delete request")
			}
		default:
			// Do nothing
		}
	}
	return tx.Commit()
}

// Close implements io.Close.
func (a *sqliteDBAccess) Close() error {
	if a.cancel != nil {
		a.cancel()
	}
	if a.db != nil {
		_ = a.db.Close()
	}
	return nil
}

// Create table if not exists.
func (a *sqliteDBAccess) ensureStateTable(parentCtx context.Context, stateTableName string) error {
	exists, err := tableExists(parentCtx, a.db, stateTableName)
	if err != nil || exists {
		return err
	}

	a.logger.Infof("Creating SQLite state table '%s'", stateTableName)

	ctx, cancel := context.WithTimeout(parentCtx, operationTimeout)
	defer cancel()

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt := fmt.Sprintf(createTableTpl, stateTableName)
	_, err = tx.Exec(stmt)
	if err != nil {
		return err
	}

	stmt = fmt.Sprintf(createTableExpirationTimeIdx, stateTableName, stateTableName)
	_, err = tx.Exec(stmt)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Check if table exists.
func tableExists(parentCtx context.Context, db *sql.DB, tableName string) (bool, error) {
	ctx, cancel := context.WithTimeout(parentCtx, operationTimeout)
	defer cancel()

	var exists string
	// Returns 1 or 0 as a string if the table exists or not.
	err := db.QueryRowContext(ctx, tableExistsStmt, tableName).Scan(&exists)
	return exists == "1", err
}

func (a *sqliteDBAccess) setValue(tx *sql.Tx, req *state.SetRequest) error {
	r, err := prepareSetRequest(a, tx, req)
	if err != nil {
		return err
	}

	hasUpdate, err := r.setValue()
	if err != nil {
		if req.ETag != nil && *req.ETag != "" {
			return state.NewETagError(state.ETagMismatch, err)
		}
		return err
	}

	if !hasUpdate {
		return fmt.Errorf("no item was updated")
	}
	return nil
}

func (a *sqliteDBAccess) deleteValue(tx *sql.Tx, req *state.DeleteRequest) error {
	r, err := prepareDeleteRequest(a, tx, req)
	if err != nil {
		return err
	}

	hasUpdate, err := r.deleteValue()
	if err != nil {
		return err
	}

	if !hasUpdate && req.ETag != nil && *req.ETag != "" {
		return state.NewETagError(state.ETagMismatch, nil)
	}
	return nil
}

func (a *sqliteDBAccess) scheduleCleanupExpiredData() {
	if a.cleanupInterval == nil {
		return
	}

	d := *a.cleanupInterval
	a.logger.Infof("Schedule expired data clean up every %d seconds", d)

	ticker := time.NewTicker(d)
	go func() {
		for {
			select {
			case <-ticker.C:
				a.cleanupTimeout()
			case <-a.ctx.Done():
				return
			}
		}
	}()
}

func (a *sqliteDBAccess) cleanupTimeout() {
	a.lock.Lock()
	defer a.lock.Unlock()

	ctx, cancel := context.WithTimeout(a.ctx, operationTimeout)
	defer cancel()

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.logger.Errorf("Error removing expired data: failed to begin transaction: %v", err)
		return
	}
	defer tx.Rollback()

	stmt := fmt.Sprintf(cleanupTimeoutStmtTpl, a.tableName)
	res, err := tx.Exec(stmt)
	if err != nil {
		a.logger.Errorf("Error removing expired data: failed to execute query: %v", err)
		return
	}

	cleaned, err := res.RowsAffected()
	if err != nil {
		a.logger.Errorf("Error removing expired data: failed to count affected rows: %v", err)
		return
	}

	err = tx.Commit()
	if err != nil {
		a.logger.Errorf("Error removing expired data: failed to commit transaction: %v", err)
		return
	}

	a.logger.Debugf("Removed %d expired rows", cleaned)
}

// Returns nil duration means never cleanup expired data.
func (a *sqliteDBAccess) parseCleanupInterval(metadata state.Metadata) (*time.Duration, error) {
	s, ok := metadata.Properties[cleanupIntervalKey]
	if ok && s != "" {
		cleanupIntervalInSec, err := strconv.ParseInt(s, 10, 0)
		if err != nil {
			return nil, fmt.Errorf("illegal cleanupIntervalInSec value: %s", s)
		}

		// Non-positive value from meta means disable auto cleanup.
		if cleanupIntervalInSec > 0 {
			d := time.Duration(cleanupIntervalInSec) * time.Second
			return &d, nil
		}
	} else {
		d := defaultCleanupInternalInSec * time.Second
		return &d, nil
	}

	return nil, nil
}

// Validates an identifier, such as table or DB name.
func validIdentifier(v string) bool {
	if v == "" {
		return false
	}

	// Loop through the string as byte slice as we only care about ASCII characters
	b := []byte(v)
	for i := 0; i < len(b); i++ {
		if (b[i] >= '0' && b[i] <= '9') ||
			(b[i] >= 'a' && b[i] <= 'z') ||
			(b[i] >= 'A' && b[i] <= 'Z') ||
			b[i] == '_' {
			continue
		}
		return false
	}
	return true
}
