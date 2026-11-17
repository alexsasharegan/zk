#!/bin/sh
# Build the release archives for one version.
#
#   scripts/release-build.sh v0.1.0 [outdir]
#
# Writes one tar.gz per target and a checksums.txt to outdir (default: dist).
# The release workflow runs this for a pushed version tag; CI runs it on every
# change so it cannot rot. It publishes nothing.
set -eu

version="${1:-}"
out="${2:-dist}"

if ! printf '%s\n' "$version" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'; then
	echo "usage: $0 <version> [outdir]   (version like v0.1.0 or v0.1.0-rc.1)" >&2
	exit 2
fi

targets="darwin/arm64 darwin/amd64 linux/arm64 linux/amd64"

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

mkdir -p "$out"
out="$(cd "$out" && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

for target in $targets; do
	os="${target%/*}"
	arch="${target#*/}"
	name="zk_${version}_${os}_${arch}"
	mkdir "$work/$name"

	CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build \
		-trimpath \
		-ldflags "-s -w -X github.com/alexsasharegan/zk/internal/cli.version=$version" \
		-o "$work/$name/zk" .

	cp README.md "$work/$name/"
	if [ -f LICENSE ]; then
		cp LICENSE "$work/$name/"
	fi

	tar -C "$work" -czf "$out/$name.tar.gz" "$name"
	echo "built $name.tar.gz"
done

# The binary for this machine must report the version it was built for.
host="zk_${version}_$(go env GOHOSTOS)_$(go env GOHOSTARCH)"
if [ -x "$work/$host/zk" ]; then
	got="$("$work/$host/zk" version)"
	if [ "$got" != "zk $version" ]; then
		echo "zk version printed '$got', want 'zk $version'" >&2
		exit 1
	fi
fi

cd "$out"
if command -v sha256sum >/dev/null 2>&1; then
	sha256sum zk_"${version}"_*.tar.gz >checksums.txt
else
	shasum -a 256 zk_"${version}"_*.tar.gz >checksums.txt
fi
echo "wrote checksums.txt"
