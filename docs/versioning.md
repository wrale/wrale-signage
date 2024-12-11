# Version Management

## Version Format

We use Semantic Versioning:
- Pre-1.0: `0.x.y` with breaking changes in minor version
- Post-1.0: `x.y.z` standard semver

## Build Versions

Build versions come from:
1. Git tag (preferred)
2. VERSION environment variable
3. Default in Makefile (0.0.1)

```bash
# View current version
./bin/wsignctl version

# Build with explicit version
VERSION=0.1.0 make build

# Tag and build
git tag -a v0.1.0 -m "Release 0.1.0"
make build  # Uses git tag
```

## Version Bumping

1. Update version number:
   ```bash
   # For patches
   git tag v0.1.1

   # For minor versions
   git tag v0.2.0
   ```

2. Build and test:
   ```bash
   make all
   ```

3. Push tags:
   ```bash 
   git push origin v0.1.1
   ```