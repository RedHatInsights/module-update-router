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
* `ENV`: Mode in which to operate. One of: "production" or "development"
   (default: "development")
* `DB_PATH`: Path to a SQLite database (default: ":memory:")
* `DB_DATA`: A CSV formatted string of module-name,account-id pairs with
   which to seed the database (default: "")
