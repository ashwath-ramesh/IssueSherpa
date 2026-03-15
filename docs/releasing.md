# Releasing

1. Commit and push your changes.
2. Tag a release: `git tag vX.Y.Z`
3. Push the tag: `git push origin vX.Y.Z`
4. GitHub Actions builds `darwin/linux` tarballs for `amd64/arm64`, uploads `checksums.txt`, and publishes the GitHub Release.

Install examples:

```bash
curl -fsSL https://raw.githubusercontent.com/ashwath-ramesh/IssueSherpa/master/scripts/install.sh | bash
curl -fsSL https://raw.githubusercontent.com/ashwath-ramesh/IssueSherpa/master/scripts/install.sh | VERSION=vX.Y.Z bash
```
