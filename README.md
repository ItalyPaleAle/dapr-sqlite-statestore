# SQLite3 state store for Dapr

This project contains a [pluggable component](https://docs.dapr.io/operations/components/pluggable-components/pluggable-components-overview/) for using SQLite3 as a state store with [Dapr](https://dapr.io).

## Example usage

See the [example](/example/) folder for an example app and how to run the component.

## Docker image

Published on GitHub Container Registry:

```text
ghcr.io/italypaleale/dapr-sqlite-statestore:latest
```

## Setup Dapr component

To setup a SQLite state store, create a component of type `state.sqlite`. See [this guide](https://docs.dapr.io/developing-applications/building-blocks/state-management/howto-get-save-state/) on how to create and apply a state store configuration.

```yaml
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: <NAME>
spec:
  type: state.sqlite
  version: v1
  metadata:
    - name: connectionString
      value: mysqlite.db
```

## Spec metadata fields

| Field              | Details | Example |
|--------------------| --------- | ---------|
| `connectionString` | The connection string to connect to the database. Usually, that's just the path to a file on disk. If needed, you pass a DSN with the options listed in the [docs for go-sqlite3](https://github.com/mattn/go-sqlite3#connection-string) | `path-to-db.db`<br>DSN: `file:mydb.db?immutable=1` |
