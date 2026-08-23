FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

# time is GNU time, for the peak RSS of a single run: the cgroup cannot report
# that, since containers are reused across test cases and across judgings.
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates time \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -u 1000 -s /bin/bash judge \
    && mkdir /sandbox \
    && chown judge:judge /sandbox

USER judge
WORKDIR /sandbox
