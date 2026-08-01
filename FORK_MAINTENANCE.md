# CloudPanda fork maintenance

This repository is an intentionally maintained fork of `ekristen/gcp-nuke`.

The fork currently exists because CloudPanda requires resource handlers that are not all available upstream:

- `CloudBuildTrigger`
- `DatabaseMigrationConnectionProfile`
- `DatabaseMigrationJob`

Do not replace this fork with upstream until all three handlers, their dependency ordering, and the consumer configuration have been verified against the upstream release.

## Verification contract

Every pull request and push to `main` runs:

```bash
bash scripts/verify-cloudpanda-fork.sh
```

The verification must pass all of the following:

- `go build`
- `go vet ./...`
- `go test ./resources/...`
- all three fork-required resource types are registered
- the resource type count does not fall below the last validated baseline

A green build alone is not sufficient if a required handler disappeared from the registry.

## Upstream synchronization

1. Start from the current fork `main`.
2. Merge the target upstream release tag or commit.
3. Resolve dependency changes without removing `cloudbuild` or `clouddms` support required by the fork handlers.
4. Run the fork verification script.
5. Review the upstream resource-type delta, especially newly enabled handlers that may fail on undeletable provider-managed resources.
6. Merge only after the consumer repository has an explicit enable/exclude decision for newly introduced resource types.

## Release contract

Release publication is deliberately explicit. A normal push to `main` must never publish a release.

A fork release must satisfy all of the following:

- tag format: `v<upstream-version>-cloudpanda.<revision>`
- GitHub Release is created in `CloudPanda-Inc/gcp-nuke`
- the Release is **not** marked as a prerelease
- exactly one asset name ends with `linux-amd64.tar.gz`
- the archive contains a single `gcp-nuke` executable
- the executable reports the expected fork version
- the executable passes the same resource-type contract as `main`
- GitHub `releases/latest` resolves to the new tag after publication

The non-prerelease requirement is part of the consumer API: the initializer resolves this repository's `releases/latest`. A prerelease is not returned by that endpoint and would leave the previous binary in use.

## Manual linux/amd64 build

Run from a clean checkout of the release commit:

```bash
tag=v1.13.0-cloudpanda.1
commit="$(git rev-parse HEAD)"
mkdir -p release

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath \
  -ldflags="-s -w -extldflags=-static \
    -X github.com/ekristen/gcp-nuke/pkg/common.SUMMARY=${tag} \
    -X github.com/ekristen/gcp-nuke/pkg/common.BRANCH=main \
    -X github.com/ekristen/gcp-nuke/pkg/common.VERSION=${tag} \
    -X github.com/ekristen/gcp-nuke/pkg/common.COMMIT=${commit}" \
  -o release/gcp-nuke .

tar -C release -czf "release/gcp-nuke-${tag}-linux-amd64.tar.gz" gcp-nuke
sha256sum "release/gcp-nuke-${tag}-linux-amd64.tar.gz"
```

Before publishing, execute the binary in a linux/amd64 environment and run:

```bash
./gcp-nuke --version
./gcp-nuke resource-types
```

After publishing, verify the public release metadata and download path before rebuilding the consumer image.

## Rollback

Deleting or reverting source code does not roll back a consumer that downloads `releases/latest`.

Rollback requires one of the following:

- redeploy a previously validated initializer image, or
- temporarily pin the consumer download to a previously validated release tag and digest

Do not create a new `latest` release merely to hide a broken release without recording the binary identity and rollback evidence.
