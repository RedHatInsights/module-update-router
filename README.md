module-update-router is a microservice that determines whether a client should
fetch a testing/prerelease module or a released/production module. It maintains
a list of account IDs internally and will respond to GET requests to `/api/v1/channel?module=<module-name>`
(for example, `insights-core`) with an appropriate URL fragment (either 
`/testing` or `/release`). It is worth noting that this service does not serve
the module itself; the client must know where to retrieve the module. This
service simply tells the client which module to retrieve.

# Building

`go build`

# Testing

`go test`

# Running

`go run .`

# Configuring

Configuration is done through environment variables.

* `ADDR`: Address on which the HTTP server should listen (default: ":8080")
* `MADDR`: Address on which the metrics HTTP server should listen (default:
   ":2112")
* `LOG_FORMAT`: Format of log output (either "json" or "text") (default: "text")
* `DB_DRIVER`: Database driver to use (either "pgx" or "sqlite3")
   (default: "sqlite3")
* `DATABASE_URL`: A URL forming a database connection string (i.e. "file::memory:")
* `DB_HOST`: Address of the database server (default: "localhost")
* `DB_PORT`: TCP port of the database server (default: "5432")
* `DB_NAME`: Name of the database (default: "postgres")
* `DB_USER`: Username on the database server (default: "postgres")
* `DB_PASS`: Password of the database user
