# Migration Guide: Database and Router Abstraction

This guide explains how to migrate from the direct GORM/Router dependencies to the new abstracted interfaces.

## Overview of Changes

### What was changed:
1. **Database Operations**: GORM-specific code is now abstracted behind `Database` interface
2. **Router Integration**: HTTP router dependencies are abstracted behind `Router` interface  
3. **Model Registry**: Models are now managed through a `ModelRegistry` interface
4. **Backward Compatibility**: Existing code continues to work with `NewAPIHandler()`

### Benefits:
- **Database Flexibility**: Switch between GORM, Bun, or other ORMs without code changes
- **Router Flexibility**: Use Gorilla Mux, Gin, Echo, or other routers
- **Better Testing**: Easy to mock database and router interactions
- **Cleaner Separation**: Business logic separated from ORM/router specifics

## Migration Path

### Option 1: No Changes Required (Backward Compatible)
Your existing code continues to work without any changes:

```go
// This still works exactly as before
handler := resolvespec.NewAPIHandler(db)
```

### Option 2: Gradual Migration to New API

#### Step 1: Use New Handler Constructor
```go
// Old way
handler := resolvespec.NewAPIHandler(gormDB)

// New way  
handler := resolvespec.NewHandlerWithGORM(gormDB)
```

#### Step 2: Use Interface-based Approach
```go
// Create database adapter
dbAdapter := resolvespec.NewGormAdapter(gormDB)

// Create model registry
registry := resolvespec.NewModelRegistry()

// Register your models
registry.RegisterModel("public.users", &User{})
registry.RegisterModel("public.orders", &Order{})

// Create handler
handler := resolvespec.NewHandler(dbAdapter, registry)
```

## Switching Database Backends

### From GORM to Bun
```go
// Add bun dependency first:
// go get github.com/uptrace/bun

// Old GORM setup
gormDB, _ := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
gormAdapter := resolvespec.NewGormAdapter(gormDB)

// New Bun setup  
sqlDB, _ := sql.Open("sqlite3", "test.db")
bunDB := bun.NewDB(sqlDB, sqlitedialect.New())
bunAdapter := resolvespec.NewBunAdapter(bunDB)

// Handler creation is identical
handler := resolvespec.NewHandler(bunAdapter, registry)
```

## Router Flexibility

### Current Gorilla Mux (Default)
```go
router := mux.NewRouter()
resolvespec.SetupRoutes(router, handler)
```

### BunRouter (Built-in Support)
```go
// Simple setup
router := bunrouter.New()
resolvespec.SetupBunRouterWithResolveSpec(router, handler)

// Or using adapter
routerAdapter := resolvespec.NewStandardBunRouterAdapter()
// Use routerAdapter.GetBunRouter() for the underlying router
```

### Using Router Adapters (Advanced)
```go
// For when you want router abstraction
routerAdapter := resolvespec.NewStandardRouter()
routerAdapter.RegisterRoute("/{schema}/{entity}", handlerFunc)
```

## Model Registration

### Old Way (Still Works)
```go
// Models registered through existing models package
handler.RegisterModel("public", "users", &User{})
```

### New Way (Recommended)
```go
registry := resolvespec.NewModelRegistry()
registry.RegisterModel("public.users", &User{})
registry.RegisterModel("public.orders", &Order{})

handler := resolvespec.NewHandler(dbAdapter, registry)
```

## Interface Definitions

### Database Interface
```go
type Database interface {
    NewSelect() SelectQuery
    NewInsert() InsertQuery  
    NewUpdate() UpdateQuery
    NewDelete() DeleteQuery
    // ... transaction methods
}
```

### Available Adapters
- `GormAdapter` - For GORM (ready to use)
- `BunAdapter` - For Bun (add dependency: `go get github.com/uptrace/bun`)
- Easy to create custom adapters for other ORMs

## Testing Benefits

### Before (Tightly Coupled)
```go
// Hard to test - requires real GORM setup
func TestHandler(t *testing.T) {
    db := setupRealGormDB()
    handler := resolvespec.NewAPIHandler(db)
    // ... test logic
}
```

### After (Mockable)
```go
// Easy to test - mock the interfaces
func TestHandler(t *testing.T) {
    mockDB := &MockDatabase{}
    mockRegistry := &MockModelRegistry{}
    handler := resolvespec.NewHandler(mockDB, mockRegistry)
    // ... test logic with mocks
}
```

## Breaking Changes
- **None for existing code** - Full backward compatibility maintained
- New interfaces are additive, not replacing existing APIs

## Recommended Migration Timeline
1. **Phase 1**: Use existing code (no changes needed)
2. **Phase 2**: Gradually adopt new constructors (`NewHandlerWithGORM`)
3. **Phase 3**: Move to interface-based approach when needed
4. **Phase 4**: Switch database backends if desired

## Getting Help
- Check example functions in `resolvespec.go`
- Review interface definitions in `database.go`
- Examine adapter implementations for patterns