# gorecipes
:egg: Recipe Manager written in go (port of https://github.com/kylemarsh/recipelister)

## Development
To run local dev server manually using an in-memory sqlite database:
```
go build
./gorecipes --config mem.config --debug --bootstrap
```

To build for deployment:
```
make dist
```
This builds the binary and a config file and places them in the `./dist`
subdirectory.  `make clean` removes that subdirectory. If you want to also
bootstrap a sqlite database to persist, `make sqlite` will do that and place
the database file in `./dist` as well.
## Configuration
gorecipes uses the following configuration values, which can be specified in
a JSON configuration file. The default configuration file is `gorecipes.conf`,
or another config file can be specified with the `--config` flag.

### Command-line Flags
- **--config**: specify a configuration file (default: `gorecipes.conf`)
- **--bootstrap**: bootstrap database with tables and sample data
- **--force**: force bootstrapping even if database is already populated. Be careful not to use this on a DB you care about!
- **--debug**: enable debugging output

### Configuration File Options
- **Debug**: enable debugging output, API commands, etc. Default `false`
- **DbDialect**: database type to use (`sqlite3` and `mysql`, for example)
- **DbDSN**: data source name for the db (filename or `:memory:` for sqlite; "user:password@host/db" for mysql...)
- **JwtSecret**: secret used to generate Json Web Tokens
- **Origins**: array of allowed origins for CORS. Required when `Debug` is `false`

Make accepts the following environment variables, which align with their
counterparts above.
- DEBUG
- DB_DIALECT
- DB_DSN
- JWT_SECRET
- CONFIG

## API
Example requests made with curl against development server (localhost:8080)

### Unauthenticated Requests
- List all recipes: `curl http://localhost:8080/recipes/`
- List all labels: `curl http://localhost:8080/labels/`
- Login: `curl -F"username=foo" -F"password=bar" http://localhost:8080/login/`

### Authenticated Requests
- Get full recipe (single recipe): `curl -H "x-access-token: $TOKEN" http://localhost:8080/priv/recipe/$RECIPE_ID`
- Delete recipe: `curl -X DELETE -H "x-access-token: $TOKEN" http://localhost:8080/priv/recipe/$RECIPE_ID`
- Get full recipe (all recipes): `curl -H "x-access-token: $TOKEN" http://localhost:8080/priv/recipes/`
- Update recipe: `curl -X PUT -H "x-access-token: $TOKEN" -F"title=Recipe Title" -F"body=Recipe body text" -F"activeTime=15" -F"totalTime=30" -F"new=on" http://localhost:8080/priv/recipe/$RECIPE_ID`
- Mark recipe as cooked: `curl -X PUT -H "x-access-token: $TOKEN" http://localhost:8080/priv/recipe/$RECIPE_ID/mark_cooked`
- Mark recipe as new: `curl -X PUT -H "x-access-token: $TOKEN" http://localhost:8080/priv/recipe/$RECIPE_ID/mark_new`

### Debugging Requests
- Get a signed JWT: `curl http://localhost:8080/debug/getToken/`
- Check JWT validity: `curl -H "x-access-token: $TOKEN" http://localhost:8080/debug/checkToken/`
- Get the bcrypt hash of a plaintext password: `curl -F"password=bar" http://localhost:8080/debug/hashPassword/`
