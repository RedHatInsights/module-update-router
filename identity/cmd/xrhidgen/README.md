`xrhidgen` generates X-Rh-Identity JSON objects suitable for passing into HTTP
requests to console.redhat.com services.

# Installation

```
go install github.com/redhatinsights/module-update-router/identity/cmd/xrhidgen@latest
```

# Usage

```
USAGE
  xrhidgen [flags] <subcommand>

SUBCOMMANDS
  user       generate a user identity JSON object
  internal   generate an internal identity JSON object
  system     generate a system identity JSON object
  associate  generate an associate identity JSON object

FLAGS
  -account-number 111000  set the identity.account_number field to `NUMBER`
  -auth-type ...          set the identity.authtype field to `STRING`
  -type ...               set the identity.type field to `STRING`
```

# Examples

```
$ xrhidgen user -email someuser@redhat.com
{"identity":{"type":"User","auth_type":"basic-auth","account_number":"111000","user":{"is_active":true,"locale":"en_US","is_org_admin":false,"username":"test@redhat.com","email":"someuser@redhat.com","first_name":"test","last_name":"user","is_internal":true}}}
```

```
$ xrhidgen system | base64 -w0
eyJpZGVudGl0eSI6eyJ0eXBlIjoiU3lzdGVtIiwiYXV0aF90eXBlIjoiY2VydC1hdXRoIiwiYWNjb3VudF9udW1iZXIiOiIxMTEwMDAiLCJzeXN0ZW0iOnsiY24iOiI3NjBlNGE5Yi1jMGNjLTQ1MzgtOGI4Yy0wOWQxYTYzMzVkZDIifX19Cg==
```

```
ht GET http://localhost:8080/api/module-update-router/v1/channel?module=insights-core "X-Rh-Identity: $(xrhidgen system | base64 -w0)"
```
