# Release Management

This document explains how to release new versions of GoREST plugins using the automated release scripts.

## Prerequisites

1. **GitHub CLI (gh)** must be installed and authenticated:
   ```bash
   # Install gh
   brew install gh  # macOS
   # or visit: https://cli.github.com/

   # Authenticate
   gh auth login
   ```

2. **Clean working directory** - All changes must be committed before releasing

3. **Permissions** - You must have push access to the plugin repository

## Release Scripts

### Single Plugin Release

Release a single plugin with automatic version bumping and changelog generation:

```bash
./release-plugin.sh <plugin-name> [patch|minor|major] [--dry-run]
```

**Arguments:**
- `plugin-name` - Name of the plugin (e.g., `gorest-auth`)
- `patch` - Increment patch version (0.1.0 → 0.1.1) **[default]**
- `minor` - Increment minor version (0.1.0 → 0.2.0)
- `major` - Increment major version (0.1.0 → 1.0.0)
- `--dry-run` - Preview changes without making them

**Examples:**

```bash
# Patch release (default)
./release-plugin.sh gorest-auth

# Explicit patch release
./release-plugin.sh gorest-auth patch

# Minor release (new features)
./release-plugin.sh gorest-auth minor

# Major release (breaking changes)
./release-plugin.sh gorest-rbac major

# Dry run to preview
./release-plugin.sh gorest-auth minor --dry-run
```

**What it does:**
1. Fetches latest tags
2. Calculates new version based on bump type
3. Shows commits since last release
4. Creates and pushes git tag
5. Creates GitHub release with auto-generated changelog

### Bulk Plugin Release

Release multiple or all plugins at once:

```bash
./release-all-plugins.sh [patch|minor|major] [--dry-run] [plugin1 plugin2 ...]
```

**Arguments:**
- `patch|minor|major` - Version bump type **[default: patch]**
- `--dry-run` - Preview changes without making them
- `plugin1 plugin2 ...` - Specific plugins to release (omit for all)

**Examples:**

```bash
# Patch release for all plugins
./release-all-plugins.sh

# Minor release for all plugins
./release-all-plugins.sh minor

# Patch release for specific plugins
./release-all-plugins.sh patch gorest-auth gorest-rbac gorest-status

# Major release for specific plugins (dry run)
./release-all-plugins.sh major --dry-run gorest-auth gorest-openapi

# Dry run all plugins with minor bump
./release-all-plugins.sh minor --dry-run
```

**What it does:**
1. Processes each plugin sequentially
2. Skips plugins with uncommitted changes
3. Creates tags and GitHub releases for each
4. Provides detailed summary at the end

## Semantic Versioning

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR** version (1.0.0 → 2.0.0)
  - Breaking changes
  - API changes that break backward compatibility
  - Requires user code changes

- **MINOR** version (0.1.0 → 0.2.0)
  - New features
  - Backward compatible additions
  - Deprecations (with warnings)

- **PATCH** version (0.1.0 → 0.1.1)
  - Bug fixes
  - Security patches
  - Documentation updates
  - Dependency updates

## Workflow Examples

### Regular Bug Fix Release

```bash
# 1. Make your changes
git commit -m "fix: correct validation logic"
git push

# 2. Release patch version
./release-plugin.sh gorest-auth patch
```

### New Feature Release

```bash
# 1. Make your changes
git commit -m "feat: add role hierarchy support"
git push

# 2. Release minor version
./release-plugin.sh gorest-rbac minor
```

### Synchronized Multi-Plugin Release

When updating dependencies across all plugins:

```bash
# 1. Update all plugins (using sync-versions.sh or manually)
./sync-versions.sh

# 2. Preview releases
./release-all-plugins.sh patch --dry-run

# 3. Execute releases
./release-all-plugins.sh patch
```

### Breaking Change Release

```bash
# 1. Update code and documentation
git commit -m "feat!: redesign authentication API"
git push

# 2. Release major version
./release-plugin.sh gorest-auth major

# 3. Update changelog with migration guide
```

## Release Checklist

Before releasing:

- [ ] All tests pass (`make test`)
- [ ] Code is linted (`make lint`)
- [ ] Changes are committed and pushed
- [ ] CHANGELOG.md is updated (if manually maintained)
- [ ] README.md reflects new features (if applicable)
- [ ] Breaking changes are documented
- [ ] Migration guide is written (for major versions)

## GitHub Release Features

Each release includes:

- **Version tag** (e.g., `v0.1.6`)
- **Release title** (e.g., "Release v0.1.6")
- **Auto-generated changelog**:
  - Commits since last release
  - PR links and authors
  - Categorized changes (features, fixes, etc.)
- **Release timestamp**
- **Source code archives** (zip, tar.gz)

## Troubleshooting

### "Uncommitted changes detected"

```bash
# Check what's uncommitted
git status

# Commit or stash changes
git add .
git commit -m "chore: prepare for release"
```

### "GitHub CLI is not authenticated"

```bash
gh auth login
# Follow the prompts
```

### "Failed to create tag"

The tag might already exist:

```bash
# Check existing tags
git tag -l

# Delete local tag if needed
git tag -d v0.1.6

# Delete remote tag if needed
git push origin :refs/tags/v0.1.6
```

### Release created but missing from GitHub

Check the release URL directly:
```bash
# Visit: https://github.com/nicolasbonnici/PLUGIN-NAME/releases
```

Or use gh CLI:
```bash
cd ../gorest-auth
gh release list
gh release view v0.1.6
```

## Manual Release (Without Scripts)

If you need to release manually:

```bash
cd ../gorest-auth

# Create tag
git tag -a v0.1.7 -m "Release v0.1.7"

# Push tag
git push origin v0.1.7

# Create GitHub release
gh release create v0.1.7 \
  --title "Release v0.1.7" \
  --generate-notes \
  --notes-start-tag v0.1.6
```

## Rollback a Release

If you need to rollback:

```bash
cd ../gorest-auth

# Delete GitHub release
gh release delete v0.1.7 --yes

# Delete remote tag
git push origin :refs/tags/v0.1.7

# Delete local tag
git tag -d v0.1.7
```

## Best Practices

1. **Always use --dry-run first** to preview changes
2. **Release during business hours** for immediate issue response
3. **Monitor GitHub Actions** after release for CI failures
4. **Announce major releases** to users/community
5. **Keep a consistent release cadence** (weekly/biweekly)
6. **Use conventional commits** for better changelogs
7. **Test releases locally** before bulk operations

## Related Scripts

- `./sync-versions.sh` - Update all plugins to latest gorest version
- `./run-all-plugins.sh` - Run commands across all plugins
- `./bump-version.sh` - Alternative manual bump script (deprecated)
