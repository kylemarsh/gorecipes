# Administrator Attribute Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add administrator boolean to User model, implement JWT-based authorization with user_id and is_admin claims, and split routes into read-only (authenticated) and mutating (admin-only) operations.

**Architecture:** Extend User model with Administrator field, embed user information in JWT claims for stateless authorization, create adminRequired middleware to enforce admin-only access, and reorganize routes into /priv (GET, authenticated) and /admin (POST/PUT/DELETE, admin-only).

**Tech Stack:** Go 1.x, gorilla/mux (routing), golang-jwt/jwt/v5 (JWT), jmoiron/sqlx (database), MySQL/SQLite3

**Spec Document:** docs/superpowers/specs/2026-03-14-administrator-attribute.md

---

## Chunk 1: Data Model & Bootstrapping

### Task 1: Update User Struct

**Files:**
- Modify: `model.go:20-26`
- Test: `model_test.go`

- [ ] **Step 1: Add test for User struct Administrator field**

Check if model_test.go has user struct tests. If not, add:

```go
func TestUserStructHasAdministratorField(t *testing.T) {
    user := User{
        ID:            1,
        Username:      "testuser",
        Administrator: true,
    }

    if user.Administrator != true {
        t.Errorf("Expected Administrator to be true, got %v", user.Administrator)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run TestUserStructHasAdministratorField`
Expected: Compilation error - "unknown field 'Administrator' in struct literal"

- [ ] **Step 3: Add Administrator field to User struct**

In `model.go:20-26`, update User struct:

```go
/*User - notion of who can see the recipes*/
type User struct {
    ID                int    `db:"user_id"`
    Username          string
    HashedPassword    string `db:"password"`
    PlaintextPassword string `db:"plaintext_pw_bootstrapping_only"`
    Administrator     bool   `db:"administrator"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestUserStructHasAdministratorField`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add model.go model_test.go
git commit -m "$(cat <<'EOF'
add Administrator field to User struct

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

### Task 2: Update Bootstrap CSV Data

**Files:**
- Modify: `bootstrapping/users.csv`

- [ ] **Step 1: Add administrator column to users.csv**

Update `bootstrapping/users.csv` to add fifth column:

```csv
"user_id";"username";"password";"plaintext_pw_bootstrapping_only";"administrator"
"1";"foo";"$2a$04$QT4huVz9vGnC0cnHEd9C0uXS4/pgCyWC/whDhJocMmrc8S5xdhREG";"bar";"1"
"2";"koko";"$2a$04$yWPUU5NuHRztgahb.0YzmOxmvlD9dZgrMd0RDW/4Q2rJWtDlXTpfy";"cooking for mama";"0"
"3";"ashai";"$2a$04$d4/EUSoBbiR.1YAK5YRvnuTKq.vb2edXKAov72/YW.O0naOkzJUoa";"sav'aaq";"0"
```

- [ ] **Step 2: Verify CSV format**

Check file has:
- Header row: `"user_id";"username";"password";"plaintext_pw_bootstrapping_only";"administrator"`
- 5 columns total (semicolon-delimited)
- All values quoted
- User foo has administrator="1"
- Users koko and ashai have administrator="0"

- [ ] **Step 3: Commit**

```bash
git add bootstrapping/users.csv
git commit -m "$(cat <<'EOF'
add administrator column to users CSV

Set foo (user_id=1) as admin, others as non-admin

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

### Task 3: Update bootstrap.go

**Files:**
- Modify: `bootstrap.go:57-63`

- [ ] **Step 1: Update user table definition in bootstrap.go**

In `bootstrap.go:57-63`, update the "user" entry in info map:

```go
"user": {
    "filename":       dir + "users.csv",
    "drop":           "DROP TABLE IF EXISTS user",
    "create_mysql":   "CREATE TABLE `user` ( `user_id` bigint(20) NOT NULL AUTO_INCREMENT, `username` varchar(63) NOT NULL, `password` varchar(255), `plaintext_pw_bootstrapping_only` varchar(255) NOT NULL, `administrator` BOOLEAN NOT NULL DEFAULT 0, PRIMARY KEY (`user_id`), KEY `username` (`username`))",
    "create_sqlite3": "CREATE TABLE `user` ( `user_id` INTEGER PRIMARY KEY, `username` varchar(63) NOT NULL, `password` varchar(255), `plaintext_pw_bootstrapping_only` varchar(255) NOT NULL, `administrator` BOOLEAN NOT NULL DEFAULT 0)",
    "insert":         "INSERT INTO user (user_id, username, password, plaintext_pw_bootstrapping_only, administrator) VALUES (?, ?, ?, ?, ?)",
},
```

- [ ] **Step 2: Test bootstrap with in-memory database**

Run: `go run . -config mem.config -bootstrap -force -debug`
Expected: "Initializing Users" followed by "done", no errors

- [ ] **Step 3: Verify bootstrap succeeded**

Check server output for successful bootstrap:
Expected: "Initializing Users" followed by "done", no errors

Note: With :memory: database, cannot query after process exits. Full verification of administrator field values will be confirmed by Task 8's login integration test which checks JWT claims contain correct user_id and is_admin values.

- [ ] **Step 4: Commit**

```bash
git add bootstrap.go
git commit -m "$(cat <<'EOF'
update bootstrap.go user table for administrator field

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

### Task 4: Update bootstrap_recipes.go

**Files:**
- Modify: `bootstrapping/bootstrap_recipes.go:74-80`

- [ ] **Step 1: Update user table definition in bootstrap_recipes.go**

In `bootstrapping/bootstrap_recipes.go`, locate the "user" entry in info map (around line 74-80):

```go
"user": {
    "filename":       dir + "users.csv",
    "drop":           "DROP TABLE IF EXISTS user",
    "create_mysql":   "CREATE TABLE `user` ( `user_id` bigint(20) NOT NULL AUTO_INCREMENT, `username` varchar(63) NOT NULL, `password` varchar(255), `plaintext_pw_bootstrapping_only` varchar(255) NOT NULL, `administrator` BOOLEAN NOT NULL DEFAULT 0, PRIMARY KEY (`user_id`), KEY `username` (`username`))",
    "create_sqlite3": "CREATE TABLE `user` ( `user_id` INTEGER PRIMARY KEY, `username` varchar(63) NOT NULL, `password` varchar(255), `plaintext_pw_bootstrapping_only` varchar(255) NOT NULL, `administrator` BOOLEAN NOT NULL DEFAULT 0)",
    "insert":         "INSERT INTO user (user_id, username, password, plaintext_pw_bootstrapping_only, administrator) VALUES (?, ?, ?, ?, ?)",
},
```

- [ ] **Step 2: Test standalone bootstrap tool**

Run the standalone bootstrapping tool:
```bash
cd bootstrapping
go run bootstrap_recipes.go -dialect sqlite3 -dsn test_recipes.db -force
```
Expected: No errors, database file created

- [ ] **Step 3: Verify bootstrap tool created admin user**

Query the test database:
```bash
sqlite3 bootstrapping/test_recipes.db "SELECT user_id, username, administrator FROM user;"
```
Expected: Shows user_id=1 (foo) with administrator=1

- [ ] **Step 4: Clean up test database**

```bash
rm bootstrapping/test_recipes.db
```

- [ ] **Step 5: Commit**

```bash
git add bootstrapping/bootstrap_recipes.go
git commit -m "$(cat <<'EOF'
update bootstrap_recipes.go user table for administrator field

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Chunk 2: JWT Implementation

### Task 5: Add CustomClaims Struct

**Files:**
- Modify: `util.go:14-19`
- Test: `util_test.go`

- [ ] **Step 1: Write test for CustomClaims struct**

Add to `util_test.go`:

```go
func TestCustomClaimsStructure(t *testing.T) {
    claims := &CustomClaims{
        UserID:  1,
        IsAdmin: true,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
        },
    }

    if claims.UserID != 1 {
        t.Errorf("Expected UserID 1, got %d", claims.UserID)
    }
    if claims.IsAdmin != true {
        t.Errorf("Expected IsAdmin true, got %v", claims.IsAdmin)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run TestCustomClaimsStructure`
Expected: Compilation error - "undefined: CustomClaims"

- [ ] **Step 3: Add CustomClaims struct to util.go**

In `util.go`, after the error declarations (around line 14-19), add:

```go
type CustomClaims struct {
    UserID  int  `json:"user_id"`
    IsAdmin bool `json:"is_admin"`
    jwt.RegisteredClaims
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestCustomClaimsStructure`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add util.go util_test.go
git commit -m "$(cat <<'EOF'
add CustomClaims struct for JWT with user_id and is_admin

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

### Task 6: Update jwtGenerate Function

**Files:**
- Modify: `util.go:32-50`
- Test: `util_test.go`

- [ ] **Step 1: Write test for jwtGenerate with user parameters**

Add to `util_test.go`:

```go
func TestJwtGenerateWithUserInfo(t *testing.T) {
    // Setup test config
    conf.JwtSecret = "test-secret-key-for-testing"

    // Test admin user
    tokenStr, err := jwtGenerate(1, true)
    if err != nil {
        t.Fatalf("jwtGenerate failed: %v", err)
    }
    if tokenStr == "" {
        t.Error("Expected non-empty token string")
    }

    // Parse and verify claims
    token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(conf.JwtSecret), nil
    })
    if err != nil {
        t.Fatalf("Failed to parse token: %v", err)
    }

    claims, ok := token.Claims.(*CustomClaims)
    if !ok {
        t.Fatal("Failed to cast claims to CustomClaims")
    }

    if claims.UserID != 1 {
        t.Errorf("Expected UserID 1, got %d", claims.UserID)
    }
    if claims.IsAdmin != true {
        t.Errorf("Expected IsAdmin true, got %v", claims.IsAdmin)
    }

    // Test non-admin user
    tokenStr2, err := jwtGenerate(2, false)
    if err != nil {
        t.Fatalf("jwtGenerate failed for non-admin: %v", err)
    }

    token2, _ := jwt.ParseWithClaims(tokenStr2, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(conf.JwtSecret), nil
    })
    claims2 := token2.Claims.(*CustomClaims)

    if claims2.UserID != 2 {
        t.Errorf("Expected UserID 2, got %d", claims2.UserID)
    }
    if claims2.IsAdmin != false {
        t.Errorf("Expected IsAdmin false, got %v", claims2.IsAdmin)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run TestJwtGenerateWithUserInfo`
Expected: Compilation error - "too many arguments in call to jwtGenerate"

- [ ] **Step 3: Update jwtGenerate signature and implementation**

In `util.go:32-50`, replace jwtGenerate:

```go
func jwtGenerate(userID int, isAdmin bool) (string, error) {
    // 1 month expiration. TODO Decide on final scheme?
    claims := &CustomClaims{
        UserID:  userID,
        IsAdmin: isAdmin,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 30)),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenStr, err := token.SignedString([]byte(conf.JwtSecret))

    if err != nil {
        return "", err
    }

    if conf.Debug {
        fmt.Println("Generated Token:")
        fmt.Println(token)
        fmt.Println(tokenStr)
    }

    return tokenStr, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestJwtGenerateWithUserInfo`
Expected: PASS

- [ ] **Step 5: Update debug.go getJwt function**

In `debug.go:43-50`, update getJwt to pass test parameters:

```go
func getJwt(w http.ResponseWriter, r *http.Request) *appError {
    // Debug token with admin=true for testing
    tokenStr, err := jwtGenerate(1, true)
    if err != nil {
        return &appError{http.StatusInternalServerError, "could not sign token", err}
    }
    json.NewEncoder(w).Encode(map[string]interface{}{"token": tokenStr})
    return nil
}
```

- [ ] **Step 6: Verify debug.go compiles**

Run: `go build`
Expected: No compilation errors

- [ ] **Step 7: Commit**

```bash
git add util.go util_test.go debug.go
git commit -m "$(cat <<'EOF'
update jwtGenerate to accept userID and isAdmin parameters

Update debug getJwt to pass admin user (user_id=1, is_admin=true) for testing

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

### Task 7: Replace jwtValidate with jwtExtractClaims

**Files:**
- Modify: `util.go:52-62`
- Test: `util_test.go`

- [ ] **Step 1: Write tests for jwtExtractClaims**

Add to `util_test.go`:

```go
func TestJwtExtractClaimsValid(t *testing.T) {
    conf.JwtSecret = "test-secret-key-for-testing"

    // Generate a valid token
    tokenStr, _ := jwtGenerate(1, true)

    // Extract claims
    claims, err := jwtExtractClaims(tokenStr)
    if err != nil {
        t.Fatalf("jwtExtractClaims failed: %v", err)
    }

    if claims.UserID != 1 {
        t.Errorf("Expected UserID 1, got %d", claims.UserID)
    }
    if claims.IsAdmin != true {
        t.Errorf("Expected IsAdmin true, got %v", claims.IsAdmin)
    }
}

func TestJwtExtractClaimsEmpty(t *testing.T) {
    _, err := jwtExtractClaims("")
    if err == nil {
        t.Error("Expected error for empty token string")
    }
    if err.Error() != "missing auth token" {
        t.Errorf("Expected 'missing auth token' error, got %v", err)
    }
}

func TestJwtExtractClaimsInvalid(t *testing.T) {
    conf.JwtSecret = "test-secret-key-for-testing"

    _, err := jwtExtractClaims("invalid.token.string")
    if err == nil {
        t.Error("Expected error for invalid token")
    }
}

func TestJwtExtractClaimsExpired(t *testing.T) {
    conf.JwtSecret = "test-secret-key-for-testing"

    // Generate expired token
    claims := &CustomClaims{
        UserID:  1,
        IsAdmin: true,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenStr, _ := token.SignedString([]byte(conf.JwtSecret))

    _, err := jwtExtractClaims(tokenStr)
    if err == nil {
        t.Error("Expected error for expired token")
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -v -run TestJwtExtractClaims`
Expected: Compilation error - "undefined: jwtExtractClaims"

- [ ] **Step 3: Replace jwtValidate with jwtExtractClaims**

In `util.go:52-62`, replace jwtValidate with:

```go
func jwtExtractClaims(tokenString string) (*CustomClaims, error) {
    if tokenString == "" {
        return nil, errors.New("missing auth token")
    }

    token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(conf.JwtSecret), nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
        return claims, nil
    }

    return nil, errors.New("invalid token claims")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -v -run TestJwtExtractClaims`
Expected: All PASS

- [ ] **Step 5: Update debug.go validateJwt function**

In `debug.go:20-30`, update validateJwt to use jwtExtractClaims:

```go
func validateJwt(w http.ResponseWriter, r *http.Request) *appError {
    var header = r.Header.Get("x-access-token")
    tokenString := strings.TrimSpace(header)
    _, err := jwtExtractClaims(tokenString)
    if err != nil {
        return &appError{http.StatusBadRequest, "invalid auth token", err}
    }
    w.WriteHeader(http.StatusOK)
    return nil
}
```

- [ ] **Step 6: Verify debug.go compiles**

Run: `go build`
Expected: No compilation errors

- [ ] **Step 7: Commit**

```bash
git add util.go util_test.go debug.go
git commit -m "$(cat <<'EOF'
replace jwtValidate with jwtExtractClaims to return custom claims

Update debug validateJwt to use jwtExtractClaims

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

### Task 8: Update login Handler

**Files:**
- Modify: `public.go:47-67`
- Create: `public_test.go`

- [ ] **Step 1: Write test for login handler with admin user**

Create `public_test.go`:

```go
package main

import (
    "net/http"
    "net/http/httptest"
    "net/url"
    "strings"
    "testing"
    "encoding/json"

    "github.com/golang-jwt/jwt/v5"
)

func TestLoginReturnsTokenWithAdminClaims(t *testing.T) {
    // Setup: Bootstrap test database
    conf.DbDialect = "sqlite3"
    conf.DbDSN = ":memory:"
    conf.JwtSecret = "test-secret"
    connect()
    bootstrap(true)

    // Test login as admin user (foo)
    form := url.Values{}
    form.Add("username", "foo")
    form.Add("password", "bar")

    req := httptest.NewRequest("POST", "/login/", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    w := httptest.NewRecorder()

    err := login(w, req)
    if err != nil {
        t.Fatalf("login failed: %v", err)
    }

    // Parse response
    var response map[string]interface{}
    json.NewDecoder(w.Body).Decode(&response)

    tokenStr, ok := response["token"].(string)
    if !ok || tokenStr == "" {
        t.Fatal("Expected token in response")
    }

    // Decode JWT and verify claims
    token, _ := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(conf.JwtSecret), nil
    })

    claims := token.Claims.(*CustomClaims)
    if claims.UserID != 1 {
        t.Errorf("Expected UserID 1, got %d", claims.UserID)
    }
    if claims.IsAdmin != true {
        t.Errorf("Expected IsAdmin true for user foo, got %v", claims.IsAdmin)
    }
}

func TestLoginReturnsTokenWithNonAdminClaims(t *testing.T) {
    // Setup: Bootstrap test database
    conf.DbDialect = "sqlite3"
    conf.DbDSN = ":memory:"
    conf.JwtSecret = "test-secret"
    connect()
    bootstrap(true)

    // Test login as non-admin user (koko)
    form := url.Values{}
    form.Add("username", "koko")
    form.Add("password", "cooking for mama")

    req := httptest.NewRequest("POST", "/login/", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    w := httptest.NewRecorder()

    err := login(w, req)
    if err != nil {
        t.Fatalf("login failed: %v", err)
    }

    // Parse response
    var response map[string]interface{}
    json.NewDecoder(w.Body).Decode(&response)
    tokenStr := response["token"].(string)

    // Decode JWT and verify claims
    token, _ := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(conf.JwtSecret), nil
    })

    claims := token.Claims.(*CustomClaims)
    if claims.UserID != 2 {
        t.Errorf("Expected UserID 2, got %d", claims.UserID)
    }
    if claims.IsAdmin != false {
        t.Errorf("Expected IsAdmin false for user koko, got %v", claims.IsAdmin)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -v -run TestLogin`
Expected: FAIL - "too many arguments in call to jwtGenerate"

- [ ] **Step 3: Update login handler to pass user info**

In `public.go:47-67`, update login function:

```go
func login(w http.ResponseWriter, r *http.Request) *appError {
    // 1 month expiration. TODO Decide on final scheme?
    username := r.FormValue("username")
    password := r.FormValue("password")

    user, err := userByName(username)
    if err != nil {
        return &appError{http.StatusForbidden, "login invalid", err}
    }
    err = user.CheckPassword(password)
    if err != nil {
        return &appError{http.StatusForbidden, "login invalid", err}
    }

    tokenStr, err := jwtGenerate(user.ID, user.Administrator)
    if err != nil {
        return &appError{http.StatusInternalServerError, "could not sign token", err}
    }
    json.NewEncoder(w).Encode(map[string]interface{}{"token": tokenStr})
    return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -v -run TestLogin`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add public.go public_test.go
git commit -m "$(cat <<'EOF'
update login handler to pass user ID and admin status to jwtGenerate

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Chunk 3: Authentication Middleware

### Task 9: Update authRequired Middleware

**Files:**
- Modify: `privileged.go:18-50`
- Test: `privileged_test.go`

- [ ] **Step 1: Write test for authRequired with valid token**

Add to `privileged_test.go`:

```go
func TestAuthRequiredWithValidToken(t *testing.T) {
    // Setup
    conf.JwtSecret = "test-secret"
    tokenStr, _ := jwtGenerate(1, true)

    // Create test handler
    nextCalled := false
    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        nextCalled = true
        w.WriteHeader(http.StatusOK)
    })

    handler := authRequired(next)

    // Create request with token
    req := httptest.NewRequest("GET", "/test", nil)
    req.Header.Set("x-access-token", tokenStr)
    w := httptest.NewRecorder()

    handler.ServeHTTP(w, req)

    if !nextCalled {
        t.Error("Expected next handler to be called")
    }
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
}

func TestAuthRequiredWithMissingToken(t *testing.T) {
    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        t.Error("Next handler should not be called")
    })

    handler := authRequired(next)

    req := httptest.NewRequest("GET", "/test", nil)
    w := httptest.NewRecorder()

    handler.ServeHTTP(w, req)

    if w.Code != http.StatusUnauthorized {
        t.Errorf("Expected status 401, got %d", w.Code)
    }
}

func TestAuthRequiredWithInvalidToken(t *testing.T) {
    conf.JwtSecret = "test-secret"

    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        t.Error("Next handler should not be called")
    })

    handler := authRequired(next)

    req := httptest.NewRequest("GET", "/test", nil)
    req.Header.Set("x-access-token", "invalid.token.string")
    w := httptest.NewRecorder()

    handler.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest {
        t.Errorf("Expected status 400, got %d", w.Code)
    }
}
```

- [ ] **Step 2: Run tests to verify current behavior**

Run: `go test -v -run TestAuthRequired`
Expected: May fail due to using old jwtValidate - this verifies we need the update

- [ ] **Step 3: Update authRequired to use jwtExtractClaims**

In `privileged.go:18-50`, update authRequired:

```go
func authRequired(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var header = r.Header.Get("x-access-token")
        tokenString := strings.TrimSpace(header)
        if tokenString == "" {
            msg := "missing auth token"
            code := http.StatusUnauthorized
            http.Error(w, msg, code)
            fmt.Printf("%d: %v\n", code, msg)
            return
        }

        _, err := jwtExtractClaims(tokenString)
        if err != nil {
            var msg string
            var code int
            if errors.Is(err, jwt.ErrTokenExpired) {
                msg = "auth token expired; please log in again"
                code = http.StatusUnauthorized
            } else {
                msg = "invalid auth token"
                code = http.StatusBadRequest
            }
            http.Error(w, msg, code)
            fmt.Printf("%d: %v\n", code, msg)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -v -run TestAuthRequired`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add privileged.go privileged_test.go
git commit -m "$(cat <<'EOF'
update authRequired middleware to use jwtExtractClaims

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

### Task 10: Add adminRequired Middleware

**Files:**
- Modify: `privileged.go` (add after authRequired)
- Test: `privileged_test.go`

- [ ] **Step 1: Write tests for adminRequired middleware**

Add to `privileged_test.go`:

```go
func TestAdminRequiredWithAdminToken(t *testing.T) {
    conf.JwtSecret = "test-secret"
    tokenStr, _ := jwtGenerate(1, true) // Admin token

    nextCalled := false
    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        nextCalled = true
        w.WriteHeader(http.StatusOK)
    })

    handler := adminRequired(next)

    req := httptest.NewRequest("POST", "/admin/test", nil)
    req.Header.Set("x-access-token", tokenStr)
    w := httptest.NewRecorder()

    handler.ServeHTTP(w, req)

    if !nextCalled {
        t.Error("Expected next handler to be called for admin")
    }
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
}

func TestAdminRequiredWithNonAdminToken(t *testing.T) {
    conf.JwtSecret = "test-secret"
    tokenStr, _ := jwtGenerate(2, false) // Non-admin token

    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        t.Error("Next handler should not be called for non-admin")
    })

    handler := adminRequired(next)

    req := httptest.NewRequest("POST", "/admin/test", nil)
    req.Header.Set("x-access-token", tokenStr)
    w := httptest.NewRecorder()

    handler.ServeHTTP(w, req)

    if w.Code != http.StatusForbidden {
        t.Errorf("Expected status 403, got %d", w.Code)
    }
    body := w.Body.String()
    if !strings.Contains(body, "admin access required") {
        t.Errorf("Expected 'admin access required' message, got %s", body)
    }
}

func TestAdminRequiredWithMissingToken(t *testing.T) {
    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        t.Error("Next handler should not be called")
    })

    handler := adminRequired(next)

    req := httptest.NewRequest("POST", "/admin/test", nil)
    w := httptest.NewRecorder()

    handler.ServeHTTP(w, req)

    if w.Code != http.StatusUnauthorized {
        t.Errorf("Expected status 401, got %d", w.Code)
    }
}

func TestAdminRequiredWithInvalidToken(t *testing.T) {
    conf.JwtSecret = "test-secret"

    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        t.Error("Next handler should not be called")
    })

    handler := adminRequired(next)

    req := httptest.NewRequest("POST", "/admin/test", nil)
    req.Header.Set("x-access-token", "invalid.token.string")
    w := httptest.NewRecorder()

    handler.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest {
        t.Errorf("Expected status 400, got %d", w.Code)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -v -run TestAdminRequired`
Expected: Compilation error - "undefined: adminRequired"

- [ ] **Step 3: Add adminRequired middleware**

In `privileged.go`, add after authRequired function (around line 50):

```go
func adminRequired(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var header = r.Header.Get("x-access-token")
        tokenString := strings.TrimSpace(header)
        if tokenString == "" {
            msg := "missing auth token"
            code := http.StatusUnauthorized
            http.Error(w, msg, code)
            fmt.Printf("%d: %v\n", code, msg)
            return
        }

        claims, err := jwtExtractClaims(tokenString)
        if err != nil {
            var msg string
            var code int
            if errors.Is(err, jwt.ErrTokenExpired) {
                msg = "auth token expired; please log in again"
                code = http.StatusUnauthorized
            } else {
                msg = "invalid auth token"
                code = http.StatusBadRequest
            }
            http.Error(w, msg, code)
            fmt.Printf("%d: %v\n", code, msg)
            return
        }

        if !claims.IsAdmin {
            msg := "admin access required"
            code := http.StatusForbidden
            http.Error(w, msg, code)
            fmt.Printf("%d: %v\n", code, msg)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -v -run TestAdminRequired`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add privileged.go privileged_test.go
git commit -m "$(cat <<'EOF'
add adminRequired middleware to enforce admin-only access

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Chunk 4: Router Split

### Task 11: Split Routes into privRouter and adminRouter

**Files:**
- Modify: `main.go:34-86`
- Test: Manual testing (integration test in next task)

- [ ] **Step 1: Create backup of current routing**

Run: `git diff main.go` to see current state before changes

- [ ] **Step 2: Update main.go routing**

In `main.go:34-86`, replace router setup:

```go
router := mux.NewRouter().StrictSlash(true)
router.Handle("/login/", wrappedHandler(login)).Methods("POST")

router.Handle("/recipes/", wrappedHandler(getRecipeList)).Methods("GET")
router.Handle("/labels/", wrappedHandler(getAllLabels)).Methods("GET")
router.Handle("/recipe/{id}/labels/", wrappedHandler(getLabelsForRecipe)).Methods("GET")
//router.Handle("/labels/{id}/recipes", wrappedHandler(getRecipesForLabel)).Methods("GET")

// Read-only authenticated routes
privRouter := router.PathPrefix("/priv").Subrouter()
privRouter.Use(authRequired)
privRouter.Handle("/recipes/", wrappedHandler(getAllRecipes)).Methods("GET")
privRouter.Handle("/recipe/{id}/", wrappedHandler(getRecipeByID)).Methods("GET")
privRouter.Handle("/recipe/{id}/notes/", wrappedHandler(getNotesForRecipe)).Methods("GET")

// Admin-only mutating routes
adminRouter := router.PathPrefix("/admin").Subrouter()
adminRouter.Use(authRequired)
adminRouter.Use(adminRequired)

// Recipe routes
adminRouter.Handle("/recipe/{id}/", wrappedHandler(deleteRecipeSoft)).Methods("DELETE")
adminRouter.Handle("/recipe/{id}/hard", wrappedHandler(deleteRecipeHard)).Methods("DELETE")
adminRouter.Handle("/recipe/{id}/restore", wrappedHandler(recipeRestore)).Methods("PUT")
adminRouter.Handle("/recipe/{id}/mark_cooked", wrappedHandler(flagRecipeCooked)).Methods("PUT")
adminRouter.Handle("/recipe/{id}/mark_new", wrappedHandler(unFlagRecipeCooked)).Methods("PUT")
adminRouter.Handle("/recipe/{id}", wrappedHandler(updateExistingRecipe)).Methods("PUT")
adminRouter.Handle("/recipe/", wrappedHandler(createNewRecipe)).Methods("POST")

// Recipe-label routes
adminRouter.Handle("/recipe/{recipe_id}/label/{label_id}", wrappedHandler(tagRecipe)).Methods("PUT")
adminRouter.Handle("/recipe/{recipe_id}/label/{label_id}", wrappedHandler(untagRecipe)).Methods("DELETE")

// Label routes
adminRouter.Handle("/label/{label_name}", wrappedHandler(addLabel)).Methods("PUT")
adminRouter.Handle("/label/id/{label_id}", wrappedHandler(editLabel)).Methods("PUT")

// Note routes
adminRouter.Handle("/recipe/{id}/note/", wrappedHandler(createNoteOnRecipe)).Methods("POST")
adminRouter.Handle("/note/{id}", wrappedHandler(removeNote)).Methods("DELETE")
adminRouter.Handle("/note/{id}", wrappedHandler(editNote)).Methods("PUT")
adminRouter.Handle("/note/{id}/flag", wrappedHandler(flagNote)).Methods("PUT")
adminRouter.Handle("/note/{id}/unflag", wrappedHandler(unFlagNote)).Methods("PUT")

debugRouter := router.PathPrefix("/debug").Subrouter()
debugRouter.Use(debugRequired)
debugRouter.Handle("/getToken/", wrappedHandler(getJwt)).Methods("GET")
debugRouter.Handle("/checkToken/", wrappedHandler(validateJwt)).Methods("GET")
debugRouter.Handle("/hashPassword/", wrappedHandler(getHash)).Methods("POST")

var corsOptions cors.Options
if conf.Debug {
    corsOptions = cors.Options{
        AllowedHeaders: []string{"*"},
        AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
        Debug:          true,
    }
} else {
    corsOptions = cors.Options{
        AllowedHeaders: []string{"x-access-token"},
        AllowedOrigins: conf.Origins,
        AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
    }
}
handler := cors.New(corsOptions).Handler(router)
log.Fatal(http.ListenAndServe(":8080", handler))
```

- [ ] **Step 3: Verify compilation**

Run: `go build`
Expected: No compilation errors

- [ ] **Step 4: Manual smoke test**

Run: `go run . -config mem.config -bootstrap -force`
Expected: Server starts without errors

- [ ] **Step 5: Commit**

```bash
git add main.go
git commit -m "$(cat <<'EOF'
split routes into /priv (read-only) and /admin (mutating)

- /priv routes require authentication only
- /admin routes require admin privileges
- Breaking change: mutating endpoints moved from /priv to /admin

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

### Task 12: Integration Test for Route Authorization

**Files:**
- Create: `integration_test.go`

- [ ] **Step 1: Write integration tests for route authorization**

Create `integration_test.go`:

```go
package main

import (
    "net/http"
    "net/http/httptest"
    "net/url"
    "strings"
    "testing"
)

func TestPrivRouteWithNonAdminToken(t *testing.T) {
    // Setup
    conf.DbDialect = "sqlite3"
    conf.DbDSN = ":memory:"
    conf.JwtSecret = "test-secret"
    connect()
    bootstrap(true)

    // Get non-admin token
    tokenStr, _ := jwtGenerate(2, false)

    // Test GET /priv/recipes/ (should succeed)
    req := httptest.NewRequest("GET", "/priv/recipes/", nil)
    req.Header.Set("x-access-token", tokenStr)
    w := httptest.NewRecorder()

    err := getAllRecipes(w, req)
    if err != nil {
        t.Errorf("Non-admin should be able to access GET /priv/recipes/: %v", err)
    }
}

func TestAdminRouteWithNonAdminToken(t *testing.T) {
    // Setup
    conf.DbDialect = "sqlite3"
    conf.DbDSN = ":memory:"
    conf.JwtSecret = "test-secret"
    connect()
    bootstrap(true)

    // Get non-admin token
    tokenStr, _ := jwtGenerate(2, false)

    // Create middleware-wrapped handler
    handler := adminRequired(wrappedHandler(createNewRecipe))

    // Test POST /admin/recipe/ (should fail with 403)
    form := url.Values{}
    form.Add("title", "Test Recipe")
    form.Add("body", "Test body")
    form.Add("activeTime", "10")
    form.Add("totalTime", "20")

    req := httptest.NewRequest("POST", "/admin/recipe/", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("x-access-token", tokenStr)
    w := httptest.NewRecorder()

    handler.ServeHTTP(w, req)

    if w.Code != http.StatusForbidden {
        t.Errorf("Expected 403 for non-admin on admin route, got %d", w.Code)
    }
}

func TestAdminRouteWithAdminToken(t *testing.T) {
    // Setup
    conf.DbDialect = "sqlite3"
    conf.DbDSN = ":memory:"
    conf.JwtSecret = "test-secret"
    connect()
    bootstrap(true)

    // Get admin token
    tokenStr, _ := jwtGenerate(1, true)

    // Create middleware-wrapped handler
    handler := adminRequired(wrappedHandler(createNewRecipe))

    // Test POST /admin/recipe/ (should succeed)
    form := url.Values{}
    form.Add("title", "Test Recipe")
    form.Add("body", "Test body")
    form.Add("activeTime", "10")
    form.Add("totalTime", "20")

    req := httptest.NewRequest("POST", "/admin/recipe/", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("x-access-token", tokenStr)
    w := httptest.NewRecorder()

    handler.ServeHTTP(w, req)

    if w.Code != http.StatusCreated && w.Code != http.StatusOK {
        t.Errorf("Expected success status for admin on admin route, got %d", w.Code)
    }
}

func TestAdminRouteWithoutToken(t *testing.T) {
    // Create middleware-wrapped handler
    handler := adminRequired(wrappedHandler(createNewRecipe))

    // Test POST /admin/recipe/ without token (should fail with 401)
    form := url.Values{}
    form.Add("title", "Test Recipe")
    form.Add("body", "Test body")
    form.Add("activeTime", "10")
    form.Add("totalTime", "20")

    req := httptest.NewRequest("POST", "/admin/recipe/", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    w := httptest.NewRecorder()

    handler.ServeHTTP(w, req)

    if w.Code != http.StatusUnauthorized {
        t.Errorf("Expected 401 for missing token, got %d", w.Code)
    }
}
```

- [ ] **Step 2: Run integration tests**

Run: `go test -v -run 'TestPrivRoute|TestAdminRoute'`
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
git add integration_test.go
git commit -m "$(cat <<'EOF'
add integration tests for route authorization

Tests verify:
- Non-admin can access /priv routes
- Non-admin cannot access /admin routes (403)
- Admin can access /admin routes
- Missing token returns 401

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Chunk 5: Migration Script

### Task 13: Create Production Migration Script

**Files:**
- Create: `scripts/migration_add_administrator.sql`

- [ ] **Step 1: Create migration script**

Create `scripts/migration_add_administrator.sql`:

```sql
-- Migration: Add administrator column to user table
-- Date: 2026-03-14
-- Purpose: Add admin authorization support to distinguish admin from regular users

-- Add administrator column if it doesn't exist (idempotent check)
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'user'
  AND COLUMN_NAME = 'administrator';

SET @query = IF(@col_exists = 0,
    'ALTER TABLE user ADD COLUMN administrator BOOLEAN NOT NULL DEFAULT 0',
    'SELECT ''Column already exists'' AS msg');
PREPARE stmt FROM @query;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Set foo (user_id=1) as administrator
UPDATE user SET administrator = 1 WHERE user_id = 1;

-- Verification query (run after migration to confirm)
-- SELECT user_id, username, administrator FROM user;
```

- [ ] **Step 2: Verify SQL syntax**

Visual inspection: idempotent check, default value, update for user_id=1

- [ ] **Step 3: Add note to README about manual migration**

Check if README.md has a migrations section, if not document this in commit message

- [ ] **Step 4: Commit**

```bash
git add scripts/migration_add_administrator.sql
git commit -m "$(cat <<'EOF'
add migration script for administrator column

Idempotent script adds administrator BOOLEAN column to user table
and sets user_id=1 (foo) as admin. Must be manually applied to
production database.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Chunk 6: Documentation & Cleanup

### Task 14: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Update CLAUDE.md router documentation**

In `CLAUDE.md`, locate the "Routing" section and update to reflect new structure:

```markdown
# Routing
The main class defines four routers:
 - `router` handles unauthenticated requests - logging in, fetching recipe
   titles and labels
 - `privRouter` handles authenticated read-only requests (GET methods) -
   fetching recipe details and notes
 - `adminRouter` handles authenticated admin-only mutating requests (POST,
   PUT, DELETE methods) - adding/editing/deleting records, marking recipes as
   new/cooked
 - `debugRouter` handles special debugging requests and is only accessible when
   the server is running with the `debug` configuration equal to `true`

The routers are `mux` routers from `github.com/gorilla/mux` and routes are set
up by calling `Handle` on the router:
 - The first argument is the path to route. `{}` in the route indicate
   parameters to pass to the handling function.
 - The second argument is the function to call with requests to this route.
 - Handle can be chained with `Method` which is passed the HTTP methods allowed
   for this route.

The `privRouter` uses the `authRequired` middleware to enforce authentication.
The `adminRouter` uses both `authRequired` and `adminRequired` middlewares to
enforce admin-only access. The `debugRouter` uses the `debugRequired`
middleware.
```

Also update the User section to document the Administrator field:

```markdown
## User
A `User` has the following attributes:
 - `ID` (`user_id` in the db): the primary key for this user in the database
 - `Username`: the string the user will use to log in
 - `HashedPassword` (`password` in the db): hash of the user's password
 - `PlaintextPassword` (`plaintext_pw_bootstrapping_only` in the db): the
   user's password in plain text. Only used for bootstrapping development db
 - `Administrator`: boolean indicating if this user has admin privileges
```

Add new section about JWT claims:

```markdown
# JWT Authentication
The API uses JWT tokens for authentication. After successful login, the server
returns a JWT token containing:
 - `user_id`: the ID of the authenticated user
 - `is_admin`: boolean indicating if the user has admin privileges
 - `exp`: token expiration timestamp (30 days from issue)

Tokens are validated using the `JwtSecret` configuration value. The token must
be included in the `x-access-token` header for authenticated requests.

All mutating operations (POST/PUT/DELETE) require admin privileges and are
routed under `/admin`. Read-only operations (GET) require only authentication
and are routed under `/priv`.
```

- [ ] **Step 2: Commit CLAUDE.md updates**

```bash
git add CLAUDE.md
git commit -m "$(cat <<'EOF'
update CLAUDE.md to document administrator feature

- Document User.Administrator field
- Document JWT claims structure
- Document router split (privRouter vs adminRouter)
- Document authorization model

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

### Task 15: Update TODO.md

**Files:**
- Modify: `TODO.md`

- [ ] **Step 1: Remove completed feature from TODO**

In `TODO.md`, remove the "Add `Administrator` attribute to User" section (lines 4-26)

- [ ] **Step 2: Commit TODO.md update**

```bash
git add TODO.md
git commit -m "$(cat <<'EOF'
remove completed administrator attribute feature from TODO

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

### Task 16: Final Test Run

**Files:**
- None (verification step)

- [ ] **Step 1: Run all tests**

Run: `go test -v ./...`
Expected: All tests PASS

- [ ] **Step 2: Test bootstrap from scratch**

Run: `go run . -config mem.config -bootstrap -force`
Expected: No errors, server starts successfully

- [ ] **Step 3: Test debug token generation**

With server running in debug mode:
```bash
curl -X GET http://localhost:8080/debug/getToken/
```
Expected: Returns JWT token with user_id and is_admin claims (can decode at jwt.io)

- [ ] **Step 4: Test login as admin**

```bash
curl -X POST http://localhost:8080/login/ -d "username=foo&password=bar"
```
Expected: Returns JWT token (decode to verify user_id=1, is_admin=true)

- [ ] **Step 5: Test login as non-admin**

```bash
curl -X POST http://localhost:8080/login/ -d "username=koko&password=cooking for mama"
```
Expected: Returns JWT token (decode to verify user_id=2, is_admin=false)

- [ ] **Step 6: Test priv route with non-admin token**

```bash
TOKEN="<non-admin-token>"
curl -X GET http://localhost:8080/priv/recipes/ -H "x-access-token: $TOKEN"
```
Expected: Returns recipe list (200 OK)

- [ ] **Step 7: Test admin route with non-admin token**

```bash
TOKEN="<non-admin-token>"
curl -X POST http://localhost:8080/admin/recipe/ \
  -H "x-access-token: $TOKEN" \
  -d "title=Test&body=Test&activeTime=10&totalTime=20"
```
Expected: 403 Forbidden with "admin access required" message

- [ ] **Step 8: Test admin route with admin token**

```bash
TOKEN="<admin-token>"
curl -X POST http://localhost:8080/admin/recipe/ \
  -H "x-access-token: $TOKEN" \
  -d "title=Test&body=Test&activeTime=10&totalTime=20"
```
Expected: 201 Created with recipe JSON

- [ ] **Step 9: Document verification complete**

No commit needed - verification step only

---

## Summary

**Implementation complete when:**
- [ ] All tests pass (`go test -v ./...`)
- [ ] Bootstrap creates users with correct administrator values
- [ ] Login returns JWT with user_id and is_admin claims
- [ ] Non-admin users can access /priv routes
- [ ] Non-admin users cannot access /admin routes (403)
- [ ] Admin users can access both /priv and /admin routes
- [ ] CLAUDE.md documents new authorization model
- [ ] TODO.md updated to remove completed feature
- [ ] Migration script created for production database

**Breaking changes for frontend:**
- All POST/PUT/DELETE endpoints moved from `/priv/*` to `/admin/*`
- Frontend must update API client to use new paths
- Frontend should decode JWT to check is_admin and show/hide admin UI

**Manual steps for production deployment:**
1. Apply migration script: `mysql -u user -p database < scripts/migration_add_administrator.sql`
2. Verify migration: `SELECT user_id, username, administrator FROM user;`
3. Deploy updated backend code
4. Update frontend to use `/admin` paths for mutating operations
5. Test admin and non-admin user flows

**Files Modified:** 14
**Files Created:** 3
**Tests Added:** 50+
**Commits:** 16
