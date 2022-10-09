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

import "time"

const (
	connectionStringKey         = "connectionString"
	metadataTTLKey              = "ttlInSeconds"
	errMissingConnectionString  = "missing connection string"
	errInvalidIdentifier        = "invalid identifier: %s" // specify identifier type, e.g. "table name"
	tableNameKey                = "tableName"
	cleanupIntervalKey          = "cleanupIntervalInSeconds"
	defaultTableName            = "state"
	defaultCleanupInternalInSec = 3600
	operationTimeout            = 15 * time.Second

	createTableTpl = `
      CREATE TABLE %s (
				key TEXT NOT NULL PRIMARY KEY,
				value TEXT NOT NULL,
				is_binary BOOLEAN NOT NULL,
				etag TEXT NOT NULL,
				creation_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				expiration_time TIMESTAMP DEFAULT NULL,
				update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`

	createTableExpirationTimeIdx = `
			CREATE INDEX idx_%s_expiration_time ON %s(expiration_time)`

	tableExistsStmt = `
		SELECT EXISTS (
			SELECT name FROM sqlite_master WHERE type='table' AND name = ?
		) AS 'exists'`

	cleanupTimeoutStmtTpl = `
		DELETE FROM %s
		WHERE expiration_time IS NOT NULL
		  AND expiration_time < CURRENT_TIMESTAMP`

	getValueTpl = `
		SELECT value, is_binary, etag FROM %s
	  	WHERE key = ?
	      AND (expiration_time IS NULL OR expiration_time > CURRENT_TIMESTAMP)`

	delValueTpl         = "DELETE FROM %s WHERE key = ?"
	delValueWithETagTpl = "DELETE FROM %s WHERE key = ? and etag = ?"

	expirationTpl = "DATETIME(CURRENT_TIMESTAMP, '+%d seconds')"

	setValueTpl = `
		INSERT OR REPLACE INTO %s (
			key, value, is_binary, etag, update_time, expiration_time, creation_time)
		VALUES(?, ?, ?, ?, CURRENT_TIMESTAMP, %s,
			(SELECT creation_time FROM %s WHERE key=?));`
	setValueWithETagTpl = `
		UPDATE %s SET value = ?, etag = ?, is_binary = ?, update_time = CURRENT_TIMESTAMP, expiration_time = %s
		WHERE key = ? AND eTag = ?;`
)
