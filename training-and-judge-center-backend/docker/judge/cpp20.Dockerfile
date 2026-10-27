FROM judge-runner:base

USER root

RUN apt-get update \
    && apt-get install -y --no-install-recommends g++ \
    && rm -rf /var/lib/apt/lists/*

# Vendored rather than downloaded, so the image builds reproducibly and offline.
# See third_party/testlib/README.md for the pinned version and how to verify it.
COPY --chmod=644 third_party/testlib/testlib.h /usr/local/include/testlib.h

USER judge
