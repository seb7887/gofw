# GitHub Actions Workflows

This directory contains automated workflows for the gofw repository.

## Auto Release Workflow

**File**: `workflows/release.yml`

### Overview

Automatically creates releases for changed Go modules on every push to `main` branch.

### How It Works

1. **Trigger**: Any push to `main` branch
2. **Detection**: Identifies which modules have changed (compares with previous commit)
3. **Versioning**: Auto-increments patch version for each changed module
4. **Testing**: Runs tests for each module (fails if tests don't pass)
5. **Release**: Creates Git tag and GitHub release with changelog

### Module Versioning

Each module has independent version tags:

```
cfgmng/v1.0.0
eventbus/v1.0.3
ginsrv/v1.0.0
httpx/v0.1.0
idgen/v1.0.0
sietch/v1.0.6
wp/v1.0.0
```

### Version Strategy

- **Automatic**: Patch version auto-increments on every change
- **Format**: `<module>/v<major>.<minor>.<patch>`
- **First release**: Starts at `v0.1.0`
- **Subsequent releases**: Increments patch (e.g., `v0.1.0` → `v0.1.1`)

### Installation After Release

Once a release is created, users can install with:

```bash
# Install specific version
go get github.com/seb7887/gofw/httpx@v0.1.2

# Install latest version
go get github.com/seb7887/gofw/httpx@latest
```

### What Gets Released

A module is released if:
- ✅ Any `.go` files in the module directory changed
- ✅ The module's `go.mod` file changed
- ❌ Only test files changed (can be configured)
- ❌ Only documentation changed (can be configured)

### Release Contents

Each release includes:
- **Git Tag**: `<module>/<version>`
- **Release Title**: `<module> <version>`
- **Changelog**: List of commits since last release
- **Install Instructions**: `go get` command
- **Module Link**: Link to module documentation

### Scripts

**Location**: `scripts/`

1. **detect-changed-modules.sh**
   - Detects which modules changed between commits
   - Outputs JSON array for matrix strategy

2. **get-module-version.sh**
   - Gets latest tag for a module
   - Auto-increments patch version
   - Handles first release (v0.1.0)

3. **generate-changelog.sh**
   - Generates markdown changelog
   - Lists commits since last tag
   - Includes module info and install command

4. **run-module-tests.sh**
   - Runs `go test` for a module
   - Handles modules without tests
   - Exits with error if tests fail

### Workflow Jobs

#### Job 1: detect-changes
- Runs on every push to main
- Detects changed modules
- Outputs list for matrix strategy
- Skips release if no changes

#### Job 2: release (matrix)
- Runs for each changed module
- Sets up Go environment
- Runs tests (fails if not passing)
- Creates tag and GitHub release
- Runs in parallel for multiple modules

### Example Workflow Run

**Scenario**: Push commits that modify `httpx/` and `sietch/`

```
1. Push to main detected
2. Detect changes:
   ✓ httpx - CHANGED
   ✓ sietch - CHANGED
   ✗ eventbus - no changes
3. Release httpx:
   - Last version: httpx/v0.1.0
   - New version: httpx/v0.1.1
   - Tests: ✅ PASSED
   - Tag created: httpx/v0.1.1
   - Release created: ✅
4. Release sietch:
   - Last version: sietch/v1.0.5
   - New version: sietch/v1.0.6
   - Tests: ✅ PASSED
   - Tag created: sietch/v1.0.6
   - Release created: ✅
5. Summary: 2 releases created
```

### Manual Version Control

To manually create a release with a specific version:

```bash
# Disable automatic workflow temporarily
# Create tag manually
git tag httpx/v1.0.0
git push origin httpx/v1.0.0

# Re-enable automatic workflow
```

### Troubleshooting

**Tests failing?**
- Check test output in workflow logs
- Fix tests and push again
- Workflow will retry automatically

**No release created?**
- Check if module actually changed
- Verify changes are in `.go` or `go.mod` files
- Check workflow run logs for detection output

**Wrong version number?**
- Check existing tags: `git tag -l "<module>/v*"`
- Manual tag might have incorrect format
- Delete and recreate: `git tag -d <tag> && git push --delete origin <tag>`

### Future Enhancements

Potential improvements to consider:

- [ ] Support for major/minor version bumps (via commit message prefixes)
- [ ] Skip release on specific commit messages (e.g., `[skip-release]`)
- [ ] Automated CHANGELOG.md file updates
- [ ] Release notes templates per module
- [ ] Slack/Discord notifications on release
- [ ] Pre-release tags for development branches
- [ ] Automated security scanning before release
- [ ] License validation
- [ ] Go module proxy warmup

### Contributing

To modify the workflow:

1. Edit files in `.github/workflows/` or `.github/scripts/`
2. Test locally if possible (scripts can run manually)
3. Commit changes to a branch
4. Create PR for review
5. Merge to main when approved

### License

Same as parent repository (MIT)
