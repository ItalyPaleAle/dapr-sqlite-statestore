# Running the example

Requirements:

- Make sure you have Dapr and the Dapr CLI installed. See Dapr's [Get started](https://docs.dapr.io/getting-started/) docs. You will need Dapr 1.9 or higher.
- You also need Go 1.19 or higher.

## Run using the Docker image

> **IMPORTANT:** This only works on Linux (Docker Desktop for Mac or Windows is not supported)

First, start the Dapr component. We will use Docker for that:

```sh
docker run -d \
  --name mysqlite-component \
  --volume /tmp/dapr-components-sockets:/tmp/dapr-components-sockets \
  ghcr.io/italypaleale/dapr-sqlite-statestore:edge
```

From this `example` folder, run:

```sh
# cd example

dapr run \
  --app-id myapp \
  --components-path ./components \
  --log-level debug \
  --\
      go run main.go
```

## Run the component as a standlone binary

> Use this while testing on Mac or Windows

First, start the Dapr component. In the root folder of this project (one level up from here), run:

```sh
mkdir -p /tmp/dapr-components-sockets/
go run .
```

Then, in **another terminal** from this `example` folder, run:

```sh
# cd example

dapr run \
  --app-id myapp \
  --components-path ./components \
  --log-level debug \
  --\
      go run main.go
```
