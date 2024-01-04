# Run a local server

```
podman run -it -d -e POSTGRES_PASSWORD=postgres postgres:latest
go run ./ -path-prefix /api -app-name module-update-router -db-driver pgx -db-pass postgres -log-level debug -seed-path seed.sql
```

# Send HTTP requests

```
ht POST http://localhost:8080/api/module-update-router/v1/event X-Rh-Identity:$(echo '{ "identity": { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }' | base64 -w 0) phase=pre_update started_at=$(date --iso-8601=seconds --utc) exit:=1 ended_at=$(date --iso-8601=seconds --utc) machine_id=$(uuidgen) core_version=3.0.156 core_path=/etc/insights-client/rpm.egg
```
