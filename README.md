# gorecipes
:egg: Recipe Manager written in go (port of https://github.com/kylemarsh/recipelister)

To run local dev server:
```
go build
./gorecipes --config mem.config --debug --bootstrap
```
To make sample requests against local server:
```
// Unauthenticated GET:
curl http://localhost:8080/recipes/
// Get an signed JWT:
curl http://localhost:8080/getToken/
// Authenticated GET:
curl -H "x-access-token: $TOKEN" http://localhost:8080/checkToken/
// Authenticated DELETE:
curl -X DELETE -H "x-access-token: $TOKEN" http://localhost:8080/checkToken/
```

