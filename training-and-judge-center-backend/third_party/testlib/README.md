# testlib

Vendored so the C++ runner image builds reproducibly and offline. Downloading it
at build time was rejected: `images-configmap.yaml` pins the runner images
because "the toolchain must not change under a contest in progress, or verdicts
could move", and a build-time fetch would make two builds of the same
`RUNNER_VERSION` produce different images.

| | |
|---|---|
| Upstream | https://github.com/MikeMirzayanov/testlib |
| Tag | `0.9.41` |
| Commit | `68f9f300b6abebec82d2a68d8ca04394f2664fb6` |
| `testlib.h` sha256 | `70f74c570f2b45d63086ae6e4c41bbb5c5ffd4428cba9914bbc0396d29be10d8` |
| License | MIT, see `LICENSE` |

To verify or update:

```sh
SHA=68f9f300b6abebec82d2a68d8ca04394f2664fb6
curl -fsSL "https://raw.githubusercontent.com/MikeMirzayanov/testlib/$SHA/testlib.h" | sha256sum
```

`cpp20.Dockerfile` copies `testlib.h` to `/usr/local/include`, which is where the
FHS puts headers not installed by the package manager and is on g++'s default
search path. Bumping it needs a `RUNNER_VERSION` bump and a rebuild.
