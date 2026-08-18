#!/usr/bin/env bash
set -euo pipefail

docker build -t judge-runner:base      -f docker/judge/base.Dockerfile .
docker build -t judge-runner:cpp20     -f docker/judge/cpp20.Dockerfile .
docker build -t judge-runner:java17    -f docker/judge/java17.Dockerfile .
docker build -t judge-runner:python310 -f docker/judge/python310.Dockerfile .

# Not derived from the base image: it needs no language toolchain, only our
# own static comparison binary.
docker build -t judge-runner:compare   -f docker/judge/compare.Dockerfile .

echo "All judge runner images built successfully."
