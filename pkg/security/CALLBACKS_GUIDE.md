# Security Provider Callbacks Guide

## Overview

The ResolveSpec security provider uses a **callback-based architecture** that requires you to implement three functions:

1. **AuthenticateCallback** - Extract user credentials from HTTP requests
2. **LoadColumnSecurityCallback** - Load column security rules for masking/hiding
3. **LoadRowSecurityCallback** - Load row security filters (WHERE clauses)

This design allows you to integrate the security provider with **any** authentication system and database schema.

---

## Why Callbacks?

The callback-based design provides:

✅ **Flexibility** - Works with any auth system (JWT, session, OAuth, custom)
✅ **Database Agnostic** - No assumptions about your security table schema
✅ **Testability** - Easy to mock for unit tests
✅ **Extensibility** - Add custom logic without modifying core code

---

## Quick Start

### Step 1: Implement the Three Callbacks

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/bitechdev/ResolveSpec/pkg/security"
)

// 1. Authentication: Extract user from request
func myAuthFunction(r *http.Request) (userID int, roles string, err error) {
    // Your auth logic here (JWT, session, header, etc.)
    token := r.Header.Get("Authorization")
    userID, roles, err = validateToken(token)
    return userID, roles, err
}

// 2. Column Security: Load column masking rules
func myLoadColumnSecurity(userID int, schema, tablename string) ([]security.ColumnSecurity, error) {
    // Your database query or config lookup here
    return loadColumnRulesFromDatabase(userID, schema, tablename)
}

// 3. Row Security: Load row filtering rules
func myLoadRowSecurity(userID int, schema, tablename string) (security.RowSecurity, error) {
    // Your database query or config lookup here
    return loadRowRulesFromDatabase(userID, schema, tablename)
}
```

### Step 2: Configure the Callbacks

```go
func main() {
    db := setupDatabase()
    handler := restheadspec.NewHandlerWithGORM(db)

    // Configure callbacks BEFORE SetupSecurityProvider
    security.GlobalSecurity.AuthenticateCallback = myAuthFunction
    security.GlobalSecurity.LoadColumnSecurityCallback = myLoadColumnSecurity
    security.GlobalSecurity.LoadRowSecurityCallback = myLoadRowSecurity

    // Setup security provider (validates callbacks are set)
    if err := security.SetupSecurityProvider(handler, &security.GlobalSecurity); err != nil {
        log.Fatal(err) // Fails if callbacks not configured
    }

    // Apply middleware
    router := mux.NewRouter()
    restheadspec.SetupMuxRoutes(router, handler)
    router.Use(mux.MiddlewareFunc(security.AuthMiddleware))
    router.Use(mux.MiddlewareFunc(security.SetSecurityMiddleware))

    http.ListenAndServe(":8080", router)
}
```

---

## Callback 1: AuthenticateCallback

### Function Signature

```go
func(r *http.Request) (userID int, roles string, err error)
```

### Parameters
- `r *http.Request` - The incoming HTTP request

### Returns
- `userID int` - The authenticated user's ID
- `roles string` - User's roles (comma-separated, e.g., "admin,manager")
- `err error` - Return error to reject the request (HTTP 401)

### Example Implementations

#### Simple Header-Based Auth
```go
func authenticateFromHeader(r *http.Request) (int, string, error) {
    userIDStr := r.Header.Get("X-User-ID")
    if userIDStr == "" {
        return 0, "", fmt.Errorf("X-User-ID header required")
    }

    userID, err := strconv.Atoi(userIDStr)
    if err != nil {
        return 0, "", fmt.Errorf("invalid user ID")
    }

    roles := r.Header.Get("X-User-Roles") // Optional
    return userID, roles, nil
}
```

#### JWT Token Auth
```go
import "github.com/golang-jwt/jwt/v5"

func authenticateFromJWT(r *http.Request) (int, string, error) {
    authHeader := r.Header.Get("Authorization")
    tokenString := strings.TrimPrefix(authHeader, "Bearer ")

    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return []byte(os.Getenv("JWT_SECRET")), nil
    })

    if err != nil || !token.Valid {
        return 0, "", fmt.Errorf("invalid token")
    }

    claims := token.Claims.(jwt.MapClaims)
    userID := int(claims["user_id"].(float64))
    roles := claims["roles"].(string)

    return userID, roles, nil
}
```

#### Session Cookie Auth
```go
func authenticateFromSession(r *http.Request) (int, string, error) {
    cookie, err := r.Cookie("session_id")
    if err != nil {
        return 0, "", fmt.Errorf("no session cookie")
    }

    session, err := sessionStore.Get(cookie.Value)
    if err != nil {
        return 0, "", fmt.Errorf("invalid session")
    }

    return session.UserID, session.Roles, nil
}
```

---

## Callback 2: LoadColumnSecurityCallback

### Function Signature

```go
func(pUserID int, pSchema, pTablename string) ([]ColumnSecurity, error)
```

### Parameters
- `pUserID int` - The authenticated user's ID
- `pSchema string` - Database schema (e.g., "public")
- `pTablename string` - Table name (e.g., "employees")

### Returns
- `[]ColumnSecurity` - List of column security rules
- `error` - Return error if loading fails

### ColumnSecurity Structure

```go
type ColumnSecurity struct {
    Schema       string   // "public"
    Tablename    string   // "employees"
    Path         []string // ["ssn"] or ["address", "street"]
    Accesstype   string   // "mask" or "hide"

    // Masking configuration (for Accesstype = "mask")
    MaskStart    int      // Mask first N characters
    MaskEnd      int      // Mask last N characters
    MaskInvert   bool     // true = mask middle, false = mask edges
    MaskChar     string   // Character to use for masking (default "*")

    // Optional fields
    ExtraFilters map[string]string
    Control      string
    ID           int
    UserID       int
}
```

### Example Implementations

#### Load from Database
```go
func loadColumnSecurityFromDB(userID int, schema, tablename string) ([]security.ColumnSecurity, error) {
    var rules []security.ColumnSecurity

    query := `
        SELECT control, accesstype, jsonvalue
        FROM core.secacces
        WHERE rid_hub IN (
            SELECT rid_hub_parent FROM core.hub_link
            WHERE rid_hub_child = ? AND parent_hubtype = 'secgroup'
        )
        AND control ILIKE ?
    `

    rows, err := db.Query(query, userID, fmt.Sprintf("%s.%s%%", schema, tablename))
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var control, accesstype, jsonValue string
        rows.Scan(&control, &accesstype, &jsonValue)

        // Parse control: "schema.table.column"
        parts := strings.Split(control, ".")
        if len(parts) < 3 {
            continue
        }

        rule := security.ColumnSecurity{
            Schema:     schema,
            Tablename:  tablename,
            Path:       parts[2:],
            Accesstype: accesstype,
        }

        // Parse JSON configuration
        var config map[string]interface{}
        json.Unmarshal([]byte(jsonValue), &config)
        if start, ok := config["start"].(float64); ok {
            rule.MaskStart = int(start)
        }
        if end, ok := config["end"].(float64); ok {
            rule.MaskEnd = int(end)
        }
        if char, ok := config["char"].(string); ok {
            rule.MaskChar = char
        }

        rules = append(rules, rule)
    }

    return rules, nil
}
```

#### Load from Static Config
```go
func loadColumnSecurityFromConfig(userID int, schema, tablename string) ([]security.ColumnSecurity, error) {
    // Define security rules in code
    allRules := map[string][]security.ColumnSecurity{
        "public.employees": {
            {
                Schema:     "public",
                Tablename:  "employees",
                Path:       []string{"ssn"},
                Accesstype: "mask",
                MaskStart:  5,
                MaskChar:   "*",
            },
            {
                Schema:     "public",
                Tablename:  "employees",
                Path:       []string{"salary"},
                Accesstype: "hide",
            },
        },
    }

    key := fmt.Sprintf("%s.%s", schema, tablename)
    rules, ok := allRules[key]
    if !ok {
        return []security.ColumnSecurity{}, nil // No rules
    }

    return rules, nil
}
```

### Column Security Examples

**Mask SSN (show last 4 digits):**
```go
ColumnSecurity{
    Path:       []string{"ssn"},
    Accesstype: "mask",
    MaskStart:  5,      // Mask first 5 characters
    MaskEnd:    0,      // Keep last 4 visible
    MaskChar:   "*",
}
// Result: "123-45-6789" → "*****6789"
```

**Hide entire field:**
```go
ColumnSecurity{
    Path:       []string{"salary"},
    Accesstype: "hide",
}
// Result: salary field returns 0 or empty
```

**Mask credit card (show last 4 digits):**
```go
ColumnSecurity{
    Path:       []string{"credit_card"},
    Accesstype: "mask",
    MaskStart:  12,
    MaskChar:   "*",
}
// Result: "1234-5678-9012-3456" → "************3456"
```

---

## Callback 3: LoadRowSecurityCallback

### Function Signature

```go
func(pUserID int, pSchema, pTablename string) (RowSecurity, error)
```

### Parameters
- `pUserID int` - The authenticated user's ID
- `pSchema string` - Database schema
- `pTablename string` - Table name

### Returns
- `RowSecurity` - Row security configuration
- `error` - Return error if loading fails

### RowSecurity Structure

```go
type RowSecurity struct {
    Schema    string // "public"
    Tablename string // "orders"
    UserID    int    // Current user ID
    Template  string // WHERE clause template (e.g., "user_id = {UserID}")
    HasBlock  bool   // If true, block ALL access to this table
}
```

### Template Variables

You can use these placeholders in the `Template` string:
- `{UserID}` - Current user's ID
- `{PrimaryKeyName}` - Primary key column name
- `{TableName}` - Table name
- `{SchemaName}` - Schema name

### Example Implementations

#### Load from Database Function
```go
func loadRowSecurityFromDB(userID int, schema, tablename string) (security.RowSecurity, error) {
    var record security.RowSecurity

    query := `
        SELECT p_template, p_block
        FROM core.api_sec_rowtemplate(?, ?, ?)
    `

    row := db.QueryRow(query, schema, tablename, userID)
    err := row.Scan(&record.Template, &record.HasBlock)
    if err != nil {
        return security.RowSecurity{}, err
    }

    record.Schema = schema
    record.Tablename = tablename
    record.UserID = userID

    return record, nil
}
```

#### Load from Static Config
```go
func loadRowSecurityFromConfig(userID int, schema, tablename string) (security.RowSecurity, error) {
    key := fmt.Sprintf("%s.%s", schema, tablename)

    // Define templates for each table
    templates := map[string]string{
        "public.orders":    "user_id = {UserID}",
        "public.documents": "user_id = {UserID} OR is_public = true",
    }

    // Define blocked tables
    blocked := map[string]bool{
        "public.admin_logs": true,
    }

    if blocked[key] {
        return security.RowSecurity{
            Schema:    schema,
            Tablename: tablename,
            UserID:    userID,
            HasBlock:  true,
        }, nil
    }

    template, ok := templates[key]
    if !ok {
        // No row security - allow all rows
        return security.RowSecurity{
            Schema:    schema,
            Tablename: tablename,
            UserID:    userID,
            Template:  "",
            HasBlock:  false,
        }, nil
    }

    return security.RowSecurity{
        Schema:    schema,
        Tablename: tablename,
        UserID:    userID,
        Template:  template,
        HasBlock:  false,
    }, nil
}
```

### Row Security Examples

**Users see only their own records:**
```go
RowSecurity{
    Template: "user_id = {UserID}",
}
// Query: SELECT * FROM orders WHERE user_id = 123
```

**Users see their records OR public records:**
```go
RowSecurity{
    Template: "user_id = {UserID} OR is_public = true",
}
```

**Complex filter with subquery:**
```go
RowSecurity{
    Template: "department_id IN (SELECT department_id FROM user_departments WHERE user_id = {UserID})",
}
```

**Block all access:**
```go
RowSecurity{
    HasBlock: true,
}
// All queries to this table will be rejected
```

---

## Complete Integration Example

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "strconv"

    "github.com/bitechdev/ResolveSpec/pkg/restheadspec"
    "github.com/bitechdev/ResolveSpec/pkg/security"
    "github.com/gorilla/mux"
    "gorm.io/gorm"
)

func main() {
    db := setupDatabase()
    handler := restheadspec.NewHandlerWithGORM(db)
    handler.RegisterModel("public", "orders", Order{})

    // ===== CONFIGURE CALLBACKS =====
    security.GlobalSecurity.AuthenticateCallback = authenticateUser
    security.GlobalSecurity.LoadColumnSecurityCallback = loadColumnSec
    security.GlobalSecurity.LoadRowSecurityCallback = loadRowSec

    // ===== SETUP SECURITY =====
    if err := security.SetupSecurityProvider(handler, &security.GlobalSecurity); err != nil {
        log.Fatal("Security setup failed:", err)
    }

    // ===== SETUP ROUTES =====
    router := mux.NewRouter()
    restheadspec.SetupMuxRoutes(router, handler)
    router.Use(mux.MiddlewareFunc(security.AuthMiddleware))
    router.Use(mux.MiddlewareFunc(security.SetSecurityMiddleware))

    log.Println("Server starting on :8080")
    http.ListenAndServe(":8080", router)
}

// Callback implementations
func authenticateUser(r *http.Request) (int, string, error) {
    userIDStr := r.Header.Get("X-User-ID")
    if userIDStr == "" {
        return 0, "", fmt.Errorf("authentication required")
    }
    userID, err := strconv.Atoi(userIDStr)
    return userID, "", err
}

func loadColumnSec(userID int, schema, table string) ([]security.ColumnSecurity, error) {
    // Your implementation here
    return []security.ColumnSecurity{}, nil
}

func loadRowSec(userID int, schema, table string) (security.RowSecurity, error) {
    return security.RowSecurity{
        Schema:    schema,
        Tablename: table,
        UserID:    userID,
        Template:  "user_id = " + strconv.Itoa(userID),
    }, nil
}
```

---

## Testing Your Callbacks

### Unit Test Example

```go
func TestAuthCallback(t *testing.T) {
    req := httptest.NewRequest("GET", "/api/orders", nil)
    req.Header.Set("X-User-ID", "123")

    userID, roles, err := myAuthFunction(req)

    assert.Nil(t, err)
    assert.Equal(t, 123, userID)
}

func TestColumnSecurityCallback(t *testing.T) {
    rules, err := myLoadColumnSecurity(123, "public", "employees")

    assert.Nil(t, err)
    assert.Greater(t, len(rules), 0)
    assert.Equal(t, "mask", rules[0].Accesstype)
}
```

---

## Common Patterns

### Pattern 1: Role-Based Security

```go
func loadColumnSec(userID int, schema, table string) ([]security.ColumnSecurity, error) {
    roles := getUserRoles(userID)

    if contains(roles, "admin") {
        // Admins see everything
        return []security.ColumnSecurity{}, nil
    }

    // Non-admins have restrictions
    return []security.ColumnSecurity{
        {Path: []string{"ssn"}, Accesstype: "mask"},
    }, nil
}
```

### Pattern 2: Tenant Isolation

```go
func loadRowSec(userID int, schema, table string) (security.RowSecurity, error) {
    tenantID := getUserTenant(userID)

    return security.RowSecurity{
        Template: fmt.Sprintf("tenant_id = %d", tenantID),
    }, nil
}
```

### Pattern 3: Caching Security Rules

```go
var securityCache = cache.New(5*time.Minute, 10*time.Minute)

func loadColumnSec(userID int, schema, table string) ([]security.ColumnSecurity, error) {
    cacheKey := fmt.Sprintf("%d:%s.%s", userID, schema, table)

    if cached, found := securityCache.Get(cacheKey); found {
        return cached.([]security.ColumnSecurity), nil
    }

    rules := loadFromDatabase(userID, schema, table)
    securityCache.Set(cacheKey, rules, cache.DefaultExpiration)

    return rules, nil
}
```

---

## Troubleshooting

### Error: "AuthenticateCallback not set"
**Solution:** Configure all three callbacks before calling `SetupSecurityProvider`:
```go
security.GlobalSecurity.AuthenticateCallback = myAuthFunc
security.GlobalSecurity.LoadColumnSecurityCallback = myColSecFunc
security.GlobalSecurity.LoadRowSecurityCallback = myRowSecFunc
```

### Error: "Authentication failed"
**Solution:** Check your `AuthenticateCallback` implementation. Ensure it returns valid user ID or proper error.

### Security rules not applying
**Solution:**
1. Check callbacks are returning data
2. Enable debug logging
3. Verify database queries return results
4. Check user has security groups assigned

---

## Next Steps

1. ✅ Implement the three callbacks for your system
2. ✅ Configure `GlobalSecurity` with your callbacks
3. ✅ Call `SetupSecurityProvider`
4. ✅ Test with different users and verify isolation
5. ✅ Review `callbacks_example.go` for more examples

For complete working examples, see:
- `pkg/security/callbacks_example.go` - 7 example implementations
- `examples/secure_server/main.go` - Full server example
- `pkg/security/README.md` - Comprehensive documentation
