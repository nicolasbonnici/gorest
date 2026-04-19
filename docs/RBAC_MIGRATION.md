# Migrating from gorest-rbac Plugin to Core Voter Layer

This guide helps you migrate from the standalone `gorest-rbac` plugin to the integrated voter layer in core gorest.

## Overview

As of gorest v1.0.0, RBAC is a **mandatory 5th hook layer** integrated into the core framework. The standalone `gorest-rbac` plugin is deprecated and will be removed in v2.0.0.

## Why Migrate?

### Benefits of Core Integration

1. **Mandatory Security**: All resources have RBAC by default
2. **Better Performance**: No plugin overhead, cached permission parsing
3. **Simpler Setup**: No plugin configuration needed
4. **Type Safety**: Compile-time checks for Authorization interface
5. **Consistent Behavior**: RBAC works the same across all resources

### Migration Timeline

- **v0.x**: gorest-rbac plugin optional
- **v1.0**: Core voter layer available, plugin deprecated
- **v1.x**: Plugin works but shows deprecation warnings
- **v2.0**: Plugin removed completely

## Migration Steps

### Step 1: Update gorest Dependency

```bash
cd your-project
go get github.com/nicolasbonnici/gorest@latest
go mod tidy
```

### Step 2: Update Configuration

**Before (plugin config):**
```yaml
plugins:
  - name: rbac
    enabled: true
    config:
      default_policy: deny_all
      superuser_role: admin
      role_hierarchy:
        admin:
          - editor
          - user
```

**After (core config):**
```yaml
rbac:
  default_policy: deny_all
  superuser_role: admin
  default_field_policy: deny
  strict_validation: false
  cache_enabled: true
  cache_ttl: 300

  role_hierarchy:
    admin:
      - editor
      - user
```

### Step 3: Update Imports

**Before:**
```go
import (
    "github.com/nicolasbonnici/gorest-rbac"
)

// Use rbac.WithRoles(ctx, roles)
```

**After:**
```go
import (
    "github.com/nicolasbonnici/gorest/rbac"
)

// Same API: rbac.WithRoles(ctx, roles)
```

### Step 4: Update Hook Implementations

**Before (plugin):**
```go
type ArticleHooks struct {
    hooks.NoOpHooks[models.Article]
}

// No authorization - plugin handled it separately
```

**After (core):**
```go
type ArticleHooks struct {
    *hooks.DefaultAuthorization[models.Article]
    hooks.NoOpHooks[models.Article]
}

func NewArticleHooks() *ArticleHooks {
    return &ArticleHooks{
        DefaultAuthorization: hooks.NewDefaultAuthorization[models.Article](
            hooks.GetGlobalRBACConfig(),
        ),
    }
}
```

### Step 5: Update Hook Factory

**Before:**
```go
func GetHooks[T Model](model T) Hooks[T] {
    switch any(model).(type) {
    case *models.Article:
        return any(&ArticleHooks{}).(Hooks[T])
    default:
        return &NoOpHooks[T]{}
    }
}
```

**After:**
```go
func GetHooks[T Model](model T) Hooks[T] {
    switch any(model).(type) {
    case *models.Article:
        return any(NewArticleHooks()).(Hooks[T])
    default:
        return hooks.NewNoOpHooks[T]()
    }
}
```

### Step 6: Update Model Tags

The tag syntax remains the same:

```go
type Article struct {
    ID      string `json:"id" db:"id" rbac:"read:*;write:*"`
    Title   string `json:"title" db:"title" rbac:"read:*;write:editor,admin"`
    Content string `json:"content" db:"content" rbac:"read:*;write:editor,admin"`
}
```

### Step 7: Remove Plugin from go.mod

```bash
# Remove the plugin dependency
go mod edit -droprequire github.com/nicolasbonnici/gorest-rbac
go mod tidy
```

## Breaking Changes

### 1. Hooks Interface Changed

**Impact**: All hook implementations must implement `Authorization[T]`

**Before:**
```go
type Hooks[T any] interface {
    StateProcessor[T]
    SQLQueryListener[T]
    SQLQueryBuilderModifier[T]
    Serializer[T]
}
```

**After:**
```go
type Hooks[T any] interface {
    Authorization[T]            // NEW - Layer 5
    StateProcessor[T]           // Layer 1
    SQLQueryListener[T]         // Layer 2
    SQLQueryBuilderModifier[T]  // Layer 3
    Serializer[T]               // Layer 4
}
```

**Fix**: Embed `DefaultAuthorization[T]` in your hooks

### 2. NoOpHooks Changed to Pointer

**Impact**: Cannot use `NoOpHooks[T]{}` anymore

**Before:**
```go
crud := NewWithHooks[MyModel](db, hooks.NoOpHooks[MyModel]{})
```

**After:**
```go
crud := NewWithHooks[MyModel](db, hooks.NewNoOpHooks[MyModel]())
```

### 3. RBAC Now Mandatory

**Impact**: All CRUD operations enforce RBAC

**Before**: RBAC only if plugin enabled
**After**: RBAC always active

**Fix**: Add `rbac:` tags to all models or use `default_field_policy: allow`

## Migration Checklist

- [ ] Update gorest to v1.0.0+
- [ ] Move `plugins.rbac` config to top-level `rbac`
- [ ] Update imports from `gorest-rbac` to `gorest/rbac`
- [ ] Add `DefaultAuthorization[T]` to all hook structs
- [ ] Change `NoOpHooks[T]{}` to `NewNoOpHooks[T]()`
- [ ] Add `rbac:` tags to all model fields
- [ ] Update hook constructors to use `NewXXXHooks()`
- [ ] Remove `gorest-rbac` from `go.mod`
- [ ] Run tests
- [ ] Update documentation

## Common Issues

### Issue: "cannot use hooks as Hooks[T]"

**Cause**: Hook struct doesn't embed `DefaultAuthorization[T]`

**Fix**:
```go
type MyHooks struct {
    *hooks.DefaultAuthorization[MyModel]  // Add this
    hooks.NoOpHooks[MyModel]
}
```

### Issue: "forbidden fields" errors everywhere

**Cause**: Models missing `rbac:` tags with deny-by-default policy

**Fix Option 1** - Add tags:
```go
type MyModel struct {
    ID   string `db:"id" rbac:"read:*;write:*"`
    Name string `db:"name" rbac:"read:*;write:user,admin"`
}
```

**Fix Option 2** - Allow by default (less secure):
```yaml
rbac:
  default_field_policy: allow
```

### Issue: All requests return 404

**Cause**: CheckRead is too restrictive or missing roles in context

**Fix**: Check middleware sets roles:
```go
ctx = rbac.WithRoles(ctx, userRoles)
```

### Issue: "method has pointer receiver"

**Cause**: Using value instead of pointer for hooks

**Fix**:
```go
// Before
return &MyHooks{...}

// After
func NewMyHooks() *MyHooks {
    return &MyHooks{
        DefaultAuthorization: hooks.NewDefaultAuthorization[MyModel](...),
    }
}
```

## Automated Migration Script

```bash
#!/bin/bash
# migrate-rbac.sh

echo "Migrating from gorest-rbac plugin to core voter layer..."

# 1. Update imports
find . -name "*.go" -type f -exec sed -i 's|github.com/nicolasbonnici/gorest-rbac|github.com/nicolasbonnici/gorest/rbac|g' {} +

# 2. Remove plugin from go.mod
go mod edit -droprequire github.com/nicolasbonnici/gorest-rbac

# 3. Update dependencies
go get github.com/nicolasbonnici/gorest@latest
go mod tidy

echo "Migration complete!"
echo ""
echo "Manual steps remaining:"
echo "1. Update hooks to embed *hooks.DefaultAuthorization[T]"
echo "2. Move rbac config from plugins to top-level in gorest.yaml"
echo "3. Add rbac tags to model fields"
echo "4. Run tests: go test ./..."
```

## Testing After Migration

```bash
# 1. Verify compilation
go build ./...

# 2. Run all tests
go test ./...

# 3. Test RBAC-specific functionality
go test ./hooks -run TestAuthorization
go test ./rbac -v

# 4. Test your resources
go test ./path/to/your/resources -v
```

## Rollback Plan

If migration fails, you can temporarily rollback:

```bash
# Revert gorest version
go get github.com/nicolasbonnici/gorest@v0.x.x

# Restore plugin
go get github.com/nicolasbonnici/gorest-rbac@latest

go mod tidy
```

## Support

If you encounter issues:

1. Check [RBAC.md](../RBAC.md) for usage guide
2. Check [AUTHORIZATION.md](../AUTHORIZATION.md) for hook reference
3. Review example code in `/examples`
4. Open an issue: https://github.com/nicolasbonnici/gorest/issues

## Comparison: Plugin vs Core

| Feature | Plugin (v0.x) | Core (v1.0+) |
|---------|---------------|--------------|
| Installation | Separate package | Built-in |
| Configuration | In `plugins` section | Top-level `rbac` section |
| Performance | Plugin overhead | Direct integration |
| Mandatory | Optional | Yes |
| Field filtering | Manual | Automatic |
| Type safety | Runtime checks | Compile-time |
| Hook integration | Separate middleware | 5th hook layer |
| Context helpers | Same API | Same API (rbac.WithRoles) |
| Role hierarchy | Supported | Supported |
| Struct tags | Same syntax | Same syntax |

## Next Steps

After successful migration:

1. Remove any plugin-specific workarounds
2. Leverage new features (automatic filtering, better errors)
3. Review security: use `default_policy: deny_all`
4. Enable caching for better performance
5. Update team documentation
