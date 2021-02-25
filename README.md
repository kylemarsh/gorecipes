# gorecipes
:egg: Recipe Manager written in go (port of https://github.com/kylemarsh/recipelister)

## Development
To run local dev server:
```
go build
./gorecipes --config mem.config --debug --bootstrap
```
## Configuration
- **Debug**: enable debugging output, API commands, etc.
- **DbDialect**: database type to use (`sqlite3` and `mysql`, for example)
- **DbDSN**: data source name for the db (filename or `:memory:` or sqlite; "user:password@host/db" for mysql...)
- **JwtSecret**: secret used to generate Json Web Tokens

## API
Example requests made with curl against development server (localhost:8080)

### Unauthenticated Requests
- List all recipes: `curl http://localhost:8080/recipes/`
- List all labels: `curl http://localhost:8080/labels/`

### Authenticated Requests
- Get full recipe (single recipe): `curl -H "x-access-token: $TOKEN" http://localhost:8080/priv/recipe/$RECIPE_ID`
- Delete recipe: `curl -X DELETE -H "x-access-token: $TOKEN" http://localhost:8080/priv/recipe/$RECIPE_ID`
- Get full recipe (all recipes): `curl -H "x-access-token: $TOKEN" http://localhost:8080/priv/recipes/`

### Debugging Requests
- Get a signed JWT: `curl http://localhost:8080/debug/getToken/`
- Check JWT validity: `curl -H "x-access-token: $TOKEN" http://localhost:8080/debug/checkToken/`
