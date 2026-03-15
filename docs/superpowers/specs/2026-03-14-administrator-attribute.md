# Administrator Attribute to User Design

## Overview
Add an `Administrator` boolean field to the User model to distinguish admin users from regular users. Split privileged routes into read-only (authenticated) and mutating (admin-only) operations. Include user_id and is_admin claims in JWT tokens to enable stateless authorization.

## Goals
- Store administrator status for each user in database
- Include user_id and is_admin in JWT claims for stateless authorization
- Split privileged routes into authenticated read-only (`/priv`) and admin-only mutating (`/admin`) operations
- Create adminRequired middleware to enforce admin-only access
- Update login response to return JWT token (client can decode to check admin status)
- Migrate production database to add administrator column
- Update bootstrapping data with user `foo` as admin

## Non-Goals
- Token invalidation/revocation system (beyond changing JwtSecret to invalidate all tokens)
- Granular role-based permissions beyond admin/non-admin
- Admin user management UI
- Audit logging of admin actions

## Database Schema

### User Table Changes
Add `administrator` column to the existing `user` table:

```sql
ALTER TABLE user ADD COLUMN administrator BOOLEAN NOT NULL DEFAULT 0;
```

**Column Specifications:**
- Type: `BOOLEAN` (MySQL: TINYINT(1), SQLite: INTEGER)
- NOT NULL with DEFAULT 0 (false/non-admin by default)
- No index needed (small table, infrequent admin checks due to JWT caching)

### Updated Schema
```sql
CREATE TABLE user (
  user_id BIGINT NOT NULL AUTO_INCREMENT,
  username VARCHAR(63) NOT NULL,
  password VARCHAR(255),
  plaintext_pw_bootstrapping_only VARCHAR(255) NOT NULL,
  administrator BOOLEAN NOT NULL DEFAULT 0,
  PRIMARY KEY (user_id),
  KEY username (username)
);
```

## Data Model

### User Struct
Update `model.go` User struct:

```go
type User struct {
    ID                int    `db:"user_id"`
    Username          string
    HashedPassword    string `db:"password"`
    PlaintextPassword string `db:"plaintext_pw_bootstrapping_only"`
    Administrator     bool
}
```

### Model Methods
No new methods needed - existing `userByName()` will automatically include Administrator field when querying.

## JWT Changes

### Custom Claims Struct
Add to `util.go`:

```go
type CustomClaims struct {
    UserID  int  `json:"user_id"`
    IsAdmin bool `json:"is_admin"`
    jwt.RegisteredClaims
}
```

**Fields:**
- `UserID` - identifies which user the token belongs to (for future audit logging)
- `IsAdmin` - cached admin status for stateless authorization
- `RegisteredClaims` - embedded standard claims (ExpiresAt, etc.)

### Update jwtGenerate
Modify signature and implementation in `util.go`:

**Old:**
```go
func jwtGenerate() (string, error)
```

**New:**
```go
func jwtGenerate(userID int, isAdmin bool) (string, error)
```

**Implementation:**
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

### Replace jwtValidate with jwtExtractClaims
Remove `jwtValidate()` and add new extraction function in `util.go`:

**Remove:**
```go
func jwtValidate(tokenString string) error
```

**Add:**
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

**Purpose:**
- Validates token signature and expiration (via jwt.ParseWithClaims)
- Extracts and returns typed CustomClaims
- Single source of truth for token validation
- Used by both authRequired and adminRequired middlewares

## Authentication Changes

### Update login Handler
Modify `login()` in `public.go`:

**Current Implementation:**
```go
func login(w http.ResponseWriter, r *http.Request) *appError {
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

    tokenStr, err := jwtGenerate()  // OLD: no parameters
    if err != nil {
        return &appError{http.StatusInternalServerError, "could not sign token", err}
    }
    json.NewEncoder(w).Encode(map[string]interface{}{"token": tokenStr})
    return nil
}
```

**Updated Implementation:**
```go
func login(w http.ResponseWriter, r *http.Request) *appError {
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

    tokenStr, err := jwtGenerate(user.ID, user.Administrator)  // NEW: pass user info
    if err != nil {
        return &appError{http.StatusInternalServerError, "could not sign token", err}
    }
    json.NewEncoder(w).Encode(map[string]interface{}{"token": tokenStr})
    return nil
}
```

**Changes:**
- Call `jwtGenerate(user.ID, user.Administrator)` with user details
- Response format unchanged (still returns `{"token": "..."}`)
- Client decodes JWT to extract `is_admin` claim

### Update authRequired Middleware
Modify `authRequired()` in `privileged.go` to use `jwtExtractClaims()`:

**Current Implementation:**
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

        _, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return []byte(conf.JwtSecret), nil
        })

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

**Updated Implementation:**
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

**Changes:**
- Replace inline `jwt.Parse()` with `jwtExtractClaims()`
- Ignore returned claims (only checking validity)
- Error handling unchanged

### Create adminRequired Middleware
Add new middleware to `privileged.go`:

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

**Purpose:**
- Extracts claims from token
- Returns 403 Forbidden if `IsAdmin` is false
- Returns 401/400 for invalid/expired tokens (same as authRequired)
- Used by `adminRouter`

## Router Split

### Current Route Organization
All authenticated routes currently under `privRouter`:
- GET routes (read-only)
- POST/PUT/DELETE routes (mutating)

### New Route Organization

**privRouter** - Read-only routes requiring authentication:
```
GET /priv/recipes/          -> getAllRecipes
GET /priv/recipe/{id}/      -> getRecipeByID
GET /priv/recipe/{id}/notes/ -> getNotesForRecipe
```

**adminRouter** - Mutating routes requiring admin privileges:
```
Recipe Management:
  DELETE /admin/recipe/{id}/           -> deleteRecipeSoft
  DELETE /admin/recipe/{id}/hard       -> deleteRecipeHard
  PUT    /admin/recipe/{id}/restore    -> recipeRestore
  PUT    /admin/recipe/{id}/mark_cooked -> flagRecipeCooked
  PUT    /admin/recipe/{id}/mark_new   -> unFlagRecipeCooked
  PUT    /admin/recipe/{id}            -> updateExistingRecipe
  POST   /admin/recipe/                -> createNewRecipe

Recipe-Label Links:
  PUT    /admin/recipe/{recipe_id}/label/{label_id}    -> tagRecipe
  DELETE /admin/recipe/{recipe_id}/label/{label_id}    -> untagRecipe

Label Management:
  PUT /admin/label/{label_name}      -> addLabel
  PUT /admin/label/id/{label_id}     -> editLabel

Note Management:
  POST   /admin/recipe/{id}/note/    -> createNoteOnRecipe
  DELETE /admin/note/{id}            -> removeNote
  PUT    /admin/note/{id}            -> editNote
  PUT    /admin/note/{id}/flag       -> flagNote
  PUT    /admin/note/{id}/unflag     -> unFlagNote
```

### Router Setup in main.go

**Current:**
```go
privRouter := router.PathPrefix("/priv").Subrouter()
privRouter.Use(authRequired)
privRouter.Handle("/recipes/", wrappedHandler(getAllRecipes)).Methods("GET")
privRouter.Handle("/recipe/{id}/", wrappedHandler(getRecipeByID)).Methods("GET")
// ... all other routes
```

**Updated:**
```go
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
```

**Key Changes:**
- `adminRouter` chains both `authRequired` and `adminRequired` middlewares
- Routes moved from `/priv/*` to `/admin/*` (breaking change for frontend)
- GET routes remain under `/priv` (backward compatible)

## Bootstrapping Data

### Update users.csv
Add `administrator` column as fifth column:

**Current:**
```csv
"user_id";"username";"password";"plaintext_pw_bootstrapping_only"
"1";"foo";"$2a$04$QT4huVz9vGnC0cnHEd9C0uXS4/pgCyWC/whDhJocMmrc8S5xdhREG";"bar"
"2";"koko";"$2a$04$yWPUU5NuHRztgahb.0YzmOxmvlD9dZgrMd0RDW/4Q2rJWtDlXTpfy";"cooking for mama"
"3";"ashai";"$2a$04$d4/EUSoBbiR.1YAK5YRvnuTKq.vb2edXKAov72/YW.O0naOkzJUoa";"sav'aaq"
```

**Updated:**
```csv
"user_id";"username";"password";"plaintext_pw_bootstrapping_only";"administrator"
"1";"foo";"$2a$04$QT4huVz9vGnC0cnHEd9C0uXS4/pgCyWC/whDhJocMmrc8S5xdhREG";"bar";"1"
"2";"koko";"$2a$04$yWPUU5NuHRztgahb.0YzmOxmvlD9dZgrMd0RDW/4Q2rJWtDlXTpfy";"cooking for mama";"0"
"3";"ashai";"$2a$04$d4/EUSoBbiR.1YAK5YRvnuTKq.vb2edXKAov72/YW.O0naOkzJUoa";"sav'aaq";"0"
```

**Format:**
- Semicolon-delimited
- Five columns: user_id, username, password, plaintext_pw_bootstrapping_only, administrator
- Administrator values: "1" (true/admin), "0" (false/regular user)
- Only user `foo` (user_id=1) set as admin
- Quoted values

### Update bootstrap_recipes.go
Modify user table definition in `bootstrapping/bootstrap_recipes.go` info map (lines 74-80):

**Current:**
```go
"user": {
    "filename":       dir + "users.csv",
    "drop":           "DROP TABLE IF EXISTS user",
    "create_mysql":   "CREATE TABLE `user` ( `user_id` bigint(20) NOT NULL AUTO_INCREMENT, `username` varchar(63) NOT NULL, `password` varchar(255), `plaintext_pw_bootstrapping_only` varchar(255) NOT NULL, PRIMARY KEY (`user_id`), KEY `username` (`username`))",
    "create_sqlite3": "CREATE TABLE `user` ( `user_id` INTEGER PRIMARY KEY, `username` varchar(63) NOT NULL, `password` varchar(255), `plaintext_pw_bootstrapping_only` varchar(255) NOT NULL)",
    "insert":         "INSERT INTO user (user_id, username, password, plaintext_pw_bootstrapping_only) VALUES (?, ?, ?, ?)",
},
```

**Updated:**
```go
"user": {
    "filename":       dir + "users.csv",
    "drop":           "DROP TABLE IF EXISTS user",
    "create_mysql":   "CREATE TABLE `user` ( `user_id` bigint(20) NOT NULL AUTO_INCREMENT, `username` varchar(63) NOT NULL, `password` varchar(255), `plaintext_pw_bootstrapping_only` varchar(255) NOT NULL, `administrator` BOOLEAN NOT NULL DEFAULT 0, PRIMARY KEY (`user_id`), KEY `username` (`username`))",
    "create_sqlite3": "CREATE TABLE `user` ( `user_id` INTEGER PRIMARY KEY, `username` varchar(63) NOT NULL, `password` varchar(255), `plaintext_pw_bootstrapping_only` varchar(255) NOT NULL, `administrator` BOOLEAN NOT NULL DEFAULT 0)",
    "insert":         "INSERT INTO user (user_id, username, password, plaintext_pw_bootstrapping_only, administrator) VALUES (?, ?, ?, ?, ?)",
},
```

CSV parsing automatically handles 5 columns (no code change needed in parsing logic).

### Update bootstrap.go
Modify embedded user CSV logic in `bootstrap.go` (similar to bootstrap_recipes.go).

Note: Need to check bootstrap.go implementation to see if it has embedded CSV or reads from file.

## Migration Script

Create `scripts/migration_add_administrator.sql` for production database:

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

**Script Properties:**
- Idempotent: Checks if column exists before adding
- Sets foo (user_id=1) as admin (assumes production has same user_ids as bootstrap data)
- All other users default to non-admin (administrator=0)

## Testing Strategy

### Unit Tests

**jwtExtractClaims (util_test.go):**
- Valid token with admin=true: returns claims with IsAdmin=true
- Valid token with admin=false: returns claims with IsAdmin=false
- Empty token string: returns error
- Invalid signature: returns error
- Expired token: returns error
- Malformed token: returns error

**jwtGenerate (util_test.go):**
- Generate token with admin=true: verify claims include IsAdmin=true and UserID
- Generate token with admin=false: verify claims include IsAdmin=false and UserID
- Verify token expiration set to 30 days

### Integration Tests

**login handler (public_test.go):**
- Login as admin user (foo): verify token contains is_admin=true
- Login as regular user (koko): verify token contains is_admin=false
- Verify token includes user_id claim

**authRequired middleware (privileged_test.go):**
- Valid token (admin): passes through
- Valid token (non-admin): passes through
- Invalid token: returns 400
- Expired token: returns 401
- Missing token: returns 401

**adminRequired middleware (privileged_test.go):**
- Valid admin token: passes through
- Valid non-admin token: returns 403
- Invalid token: returns 400
- Expired token: returns 401
- Missing token: returns 401

**Route authorization:**
- GET /priv/recipes/ with valid non-admin token: succeeds (200)
- POST /admin/recipe/ with valid admin token: succeeds (201)
- POST /admin/recipe/ with valid non-admin token: fails (403)
- POST /admin/recipe/ with no token: fails (401)

### Manual Testing
1. Bootstrap database, verify user `foo` has administrator=1
2. Login as foo, decode JWT, verify is_admin=true and user_id=1
3. Login as koko, decode JWT, verify is_admin=false and user_id=2
4. Access GET /priv/recipes/ with koko token: succeeds
5. Access POST /admin/recipe/ with koko token: fails with 403
6. Access POST /admin/recipe/ with foo token: succeeds
7. Change JwtSecret config, verify all existing tokens become invalid

## Error Handling

### Middleware Error Responses

**authRequired:**
| Error Condition | Status Code | Response Body |
|----------------|-------------|---------------|
| Missing token | 401 | "missing auth token" |
| Expired token | 401 | "auth token expired; please log in again" |
| Invalid token | 400 | "invalid auth token" |

**adminRequired:**
| Error Condition | Status Code | Response Body |
|----------------|-------------|---------------|
| Missing token | 401 | "missing auth token" |
| Expired token | 401 | "auth token expired; please log in again" |
| Invalid token | 400 | "invalid auth token" |
| Non-admin user | 403 | "admin access required" |

**login:**
| Error Condition | Status Code | Response Body |
|----------------|-------------|---------------|
| Invalid username/password | 403 | "login invalid" |
| Token generation failed | 500 | "could not sign token" |
| Success | 200 | `{"token": "..."}` |

## Implementation Notes

### Code Organization
- Model changes: `model.go` (User struct - automatic, no method changes)
- JWT changes: `util.go` (CustomClaims, jwtGenerate, jwtExtractClaims)
- Handler changes: `public.go` (login), `privileged.go` (authRequired, adminRequired)
- Routing: `main.go` (split privRouter/adminRouter)
- Bootstrap updates: `bootstrap.go`, `bootstrapping/bootstrap_recipes.go`, `bootstrapping/users.csv`
- Migration script: `scripts/migration_add_administrator.sql`

### Breaking Changes for Frontend
**Route Path Changes:**
All mutating operations moved from `/priv/*` to `/admin/*`:
- `/priv/recipe/` (POST) → `/admin/recipe/` (POST)
- `/priv/recipe/{id}` (PUT) → `/admin/recipe/{id}` (PUT)
- `/priv/recipe/{id}` (DELETE) → `/admin/recipe/{id}` (DELETE)
- etc.

**Frontend must:**
1. Update all POST/PUT/DELETE requests to use `/admin/*` prefix
2. Handle 403 responses for non-admin users attempting admin operations
3. Optionally decode JWT to show/hide admin UI elements based on `is_admin` claim

### JWT Security Considerations
- **JWTs are readable** - client can decode and see user_id and is_admin
- **JWTs are tamper-proof** - client cannot modify claims without invalidating signature
- **JWTs cannot be selectively revoked** - changing admin status requires user to re-login
- **Nuclear option available** - changing JwtSecret invalidates all tokens

### Consistency Patterns
- Middleware structure matches existing `authRequired` and `debugRequired` patterns
- Error handling via HTTP status codes and text messages
- Router setup via PathPrefix and Subrouter
- Middleware chaining via `.Use()`
- CSV format: semicolon-delimited, quoted values, header row

### Bootstrap.go Investigation Needed
The design assumes `bootstrap.go` has similar structure to `bootstrap_recipes.go`. Need to verify:
- Does it embed CSV data or read from files?
- Does it use the same info map pattern?
- Does it need the same table definition updates?

## Future Considerations

### Out of Scope
- Granular permissions (read-only admin, label-only admin, etc.)
- Audit logging of admin actions
- Admin user management endpoints
- Token refresh mechanism
- Token revocation/blacklist system

### Potential Enhancements
- Add user_id to context for use in handlers (for audit logging)
- Add "last login" timestamp to User table
- Add endpoint to check current user's admin status
- Add token refresh endpoint to extend expiration

## Acceptance Criteria

- [ ] User struct includes Administrator field with `db:"administrator"` tag
- [ ] Database schema includes administrator column (BOOLEAN, NOT NULL, DEFAULT 0)
- [ ] CustomClaims struct includes UserID and IsAdmin fields
- [ ] jwtGenerate accepts userID and isAdmin parameters
- [ ] jwtGenerate creates token with CustomClaims
- [ ] jwtExtractClaims validates token and returns CustomClaims
- [ ] jwtValidate function removed (replaced by jwtExtractClaims)
- [ ] login handler calls jwtGenerate with user.ID and user.Administrator
- [ ] login response format unchanged (returns {"token": "..."})
- [ ] authRequired middleware uses jwtExtractClaims
- [ ] adminRequired middleware created and checks claims.IsAdmin
- [ ] adminRequired returns 403 for non-admin users
- [ ] privRouter contains only GET routes
- [ ] adminRouter created with /admin prefix
- [ ] adminRouter uses both authRequired and adminRequired middlewares
- [ ] All POST/PUT/DELETE routes moved to adminRouter
- [ ] bootstrapping/users.csv includes administrator column
- [ ] User foo (user_id=1) set as administrator in CSV
- [ ] bootstrap_recipes.go table definitions updated for administrator column
- [ ] bootstrap.go table definitions updated for administrator column
- [ ] Migration script is idempotent
- [ ] Migration script sets foo (user_id=1) as administrator
- [ ] Tests cover JWT generation with admin/non-admin users
- [ ] Tests cover jwtExtractClaims error cases
- [ ] Tests cover adminRequired middleware authorization logic
- [ ] Tests verify non-admin users cannot access admin routes
- [ ] CLAUDE.md updated to reflect new router structure and authorization model
