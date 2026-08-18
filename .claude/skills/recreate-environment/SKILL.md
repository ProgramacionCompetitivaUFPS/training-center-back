---
name: recreate-environment
description: Guided step-by-step recreation of the training-center GKE environment — Terraform (cloud, capa 1) plus the app bootstrap (cluster, capa 2). Use whenever you need to stand the environment up again, for any reason — the trial expired, you want a fresh cluster (e.g. to enable a create-time feature like Dataplane V2), you're migrating to another project or account, you're onboarding a new environment, or anything else. Destructive: recreates cloud and cluster resources.
disable-model-invocation: true
---

# Recreate the training-center environment

Drive this **interactively**. For each step: state in one line what it does, run the command **showing its output**, and **ask the user to confirm before every destructive action**. Never batch destructive steps or run a teardown without an explicit yes. Adapt to what the output actually shows instead of assuming success.

The capa-2 portion (steps 6–7) is the guided equivalent of `training-and-judge-center-backend/deploy/k8s/bootstrap.ps1`. Keep the two in sync — if the manifests or their order change, update both. `training-and-judge-center-backend/GKE_DEPLOY_ROADMAP.md` can be consulted as a reference for teardown/backup commands, but it may be outdated — treat it as a starting point and contrast anything you take from it with the user before running it.

**Facts** (all paths under `training-and-judge-center-backend/`):
- Terraform (capa 1): `deploy/gcp/` · Manifests (capa 2): `deploy/k8s/`
- Project `training-center-502916` · cluster `training-center` · zone `us-east1-b` · namespace `training-center`
- Registry: `us-east1-docker.pkg.dev/training-center-502916/training-center/backend`
- KEDA: `v2.20.1`

## 0. Scope, target, and what to keep — ask first
Ask all of the following up front, and don't proceed until answered:
1. **Scope** — recreate **both layers** (capa 1 cloud + capa 2 app), or **only capa 2** (redeploy the app on the existing cluster). Only capa 2 → **skip steps 2 and 3**, and skip step 8 unless the Ingress IP actually changed.
2. **Target** — recreate in the **same project**, or **migrate to a totally different GCP account/project** (new `project_id`/`project_number` in the Terraform vars — the "trial expired, moving accounts" case). Migration builds the new environment elsewhere and **leaves the old one running until the new one is verified** (step 10) — so there is no teardown in step 2; offer it only at the end.
   **Never infer this from how the user phrased the request** (e.g. "let's test a migration," "probemos migrar") — that kind of phrasing is often exploratory, not a firm decision. Ask it as its own explicit choice, spelling out what migration concretely requires (a real, separate `project_id` with its own billing) before asking for that project's details — don't jump straight to "what's the project_id" on an assumed target.
3. **What to keep** — always ask about all three, one by one, and record each choice:
   - **Data** (Postgres + uploads bucket) — carry over (capture in step 1, restore in step 7) or start empty.
   - **Images** — same-project recreate: keep the registry (step 2 must not destroy it). Migration: carry them to the new registry (step 5 — copy the history, or rebuild from git tags).
   - **Secrets** — the default is that the user types five of the six by hand (step 6);
     `ROTATION_CACHE_ENCRYPTION_KEY` is always generated, never typed (see step 6 for why). Only
     if the user explicitly asks, reuse the current values (captured in step 1) instead.

## 1. Capture — before any teardown (whenever keeping data or secrets)
Do this while the old environment is still up, and confirm you hold everything **before** step 2:
- **Postgres** — back up and download the dump locally (restored in step 7 if keeping data).
- **Secrets** — only if the user chose to preserve them: save the secret objects **without decoding** them, so the preserved values can be reused in step 6: `kubectl get secret app-secrets keda-rabbitmq -n training-center -o yaml > <path>/captured-secrets.yaml` (keep it out of git, delete after step 6). Values stay base64 and never need typing.
- **Uploads** — same-project recreate: the bucket persists (step 2 must not destroy it). Migration: note the source bucket; its objects get copied to the new bucket once it exists (step 7).

## 2. Teardown (same-project recreate only — destructive, confirm each)
Skip entirely for a migration — the old project stays up until step 10. Otherwise **target the destroy**: never a blanket `terraform destroy`, which would also delete the registry and the uploads bucket. Destroy only the cluster and any create-time resources being recreated (`terraform destroy -target=...`, or delete the cluster out-of-band), sparing whatever the user chose to keep in step 0 (registry, uploads bucket) plus the tfstate. Confirm and delete one resource at a time, showing output.

## 3. Apply capa 1 (Terraform)
From `deploy/gcp/`: `terraform init` (if needed) → `terraform plan` (**show it and review with the user**) → `terraform apply` (confirm). Then show **and remember** the outputs — `ingress_ip`, `wif_provider`, `github_deployer_email` — so you can hand them back later when the user needs them (DNS in step 8, the GitHub secrets) without making them re-run `terraform output`.

## 4. Cluster credentials
`gcloud container clusters get-credentials training-center --zone us-east1-b`. Verify with `kubectl get nodes`.

## 5. Images
The registry is Terraform-managed and survives a normal teardown, so its `v*` images usually remain. Confirm one exists:
`gcloud artifacts docker images list us-east1-docker.pkg.dev/training-center-502916/training-center/backend --include-tags --filter="tags~^v" --limit=1`

If it's empty (registry recreated, or brand-new project), **help the user push it here — don't send them elsewhere.** One image serves both the `api` and `judge-worker` containers, so it's the only build. **First ask which code to build from:**
- **A GitHub release tag** (recommended — ties the image to real released code). List them with `git fetch --tags` then `git tag -l 'v*' --sort=-v:refname`, let the user pick one, check the working tree is clean, `git checkout <tag>`, build the image labeled with that same version, push, then return to the previous branch. This matches what the CI does when a `v*` tag is pushed.
- **The current working tree** — only if the user wants an unreleased build. Agree on a version tag; note it won't correspond to any git tag unless they create one.

Then walk them through it, adapting to their shell:
```
gcloud auth configure-docker us-east1-docker.pkg.dev --quiet
docker build -t us-east1-docker.pkg.dev/training-center-502916/training-center/backend:vX.Y.Z ./training-and-judge-center-backend
docker push us-east1-docker.pkg.dev/training-center-502916/training-center/backend:vX.Y.Z
```
This is the manual equivalent of the CI's build-and-push. The CI *deploy* job additionally runs migrations and `set image`, which fail against a brand-new cluster on its first run — that's why the initial push is manual and the app instead comes up through step 7.

**Judge language images — check these too, same condition (empty registry).** The judge-worker's `prepull-language-images` init container pulls `judge-runner-{cpp20,java17,python310,compare}` at the versions pinned in `deploy/k8s/judge/images-configmap.yaml` from this **same registry**, so they're wiped right along with the backend image. This doesn't fail loudly here — the pod schedules fine and `dind` passes, then `prepull-language-images` crash-loops on "manifest unknown" only once a judge-worker pod actually tries to start (step 10, or KEDA's first real scale-up), which makes it easy to miss until later. Check and fix it now, not then:
```
gcloud artifacts docker images list <registry> --include-tags --filter="tags~^judge-runner" --limit=1
```
If empty, rebuild and push (from `training-and-judge-center-backend/`, Docker running locally):
```
bash scripts/build-judge-images.sh
```
then tag and push the 4 sandbox images (not `base` — that's just an intermediate layer, never pushed). The versions come from `deploy/k8s/judge/images-configmap.yaml`, the single source of truth — never hardcode them:
```
$REG = "us-east1-docker.pkg.dev/training-center-502916/training-center"
$v = @{}
Select-String '^\s{2}([A-Z_]+):\s*"(.+)"$' deploy/k8s/judge/images-configmap.yaml | ForEach-Object {
  $v[$_.Matches.Groups[1].Value] = $_.Matches.Groups[2].Value
}
$images = [ordered]@{
  cpp20     = $v['RUNNER_VERSION']
  java17    = $v['RUNNER_VERSION']
  python310 = $v['RUNNER_VERSION']
  compare   = $v['COMPARE_VERSION']
}
foreach ($lang in $images.Keys) {
  docker tag  "judge-runner:$lang" "$REG/judge-runner-${lang}:$($images[$lang])"
  docker push "$REG/judge-runner-${lang}:$($images[$lang])"
}
```

**Migrating to another project?** The new registry starts empty. To bring the app up you only need the latest `vX.Y.Z` — build it from its git tag as above (git is the source of truth, so no version is really lost). To carry the **full image history** with exact digests instead of rebuilding, authenticate to both registries and copy the whole repo — e.g. `gcrane cp -r <old-registry> <new-registry>` (or pull / re-tag / push per image).

## 6. Secrets — template file the user fills, read by kubectl, never by you
Keep secret plaintext out of this conversation. **Never `Read` the filled file** — that would put the values in the transcript, no better than pasting them. Let `kubectl` consume it instead:

1. Write a template env-file with the keys and **blank values** to a scratch path — `app-secrets.env`:
   ```
   DB_PASSWORD=
   JWT_SECRET=
   RABBITMQ_PASSWORD=
   ADMIN_PASSWORD=
   SMTP_PASSWORD=
   ROTATION_CACHE_ENCRYPTION_KEY=
   ```
   and a second file `keda-rabbitmq.env` for the separate secret KEDA uses to watch the queue and autoscale the judge workers. It has one key, `host`, holding a full AMQP connection URL — put the **same** value as `RABBITMQ_PASSWORD` between `judge:` and `@`:
   ```env
   host=amqp://judge:<RABBITMQ_PASSWORD>@rabbitmq.training-center.svc.cluster.local:5672/
   ```
2. **By default the user types the first five values by hand.** Override a key's source only when the user explicitly asked for it: a secret they chose to **preserve** — you fill it from the value captured in step 1 (copied, not typed); a secret they chose to **generate** randomly — you fill it from a generator (PowerShell `Get-Random` / `openssl rand`). In both of those cases write the value straight into the file, never to stdout, and never `Read` the file back. Absent an explicit preserve-or-generate choice for a key, the user types it. Use the **same RabbitMQ password** in both files.

   **`ROTATION_CACHE_ENCRYPTION_KEY` is the one exception to "the user types it"** — unlike the other five, it isn't a human-memorable password, it's the AES-256-GCM key that encrypts the refresh-rotation cache payload in Redis (see `internal/adapter/auth/rotation_cache.go`). There's nothing meaningful for a person to type. Unless the user chose to **preserve** it (captured in step 1), always **generate** it: 32 random bytes from a *cryptographic* generator, base64-encoded — `openssl rand -base64 32`, or PowerShell (`Get-Random` is not cryptographically secure — don't use it for this one, it's fine for the other five, which are just passwords):
   ```powershell
   $bytes = [byte[]]::new(32)
   [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
   [Convert]::ToBase64String($bytes)
   ```
   Write it straight into the file like any other generated value, never to stdout. It must be **exactly** 32 raw bytes once decoded — `NewRedisRotationCache` rejects anything else at startup — so don't substitute a plain random string here, it needs to actually be base64.
3. Create the secrets, letting kubectl read the files (you never see the contents):
   - `kubectl create secret generic app-secrets -n training-center --from-env-file=<path>/app-secrets.env`
   - `kubectl create secret generic keda-rabbitmq -n training-center --from-env-file=<path>/keda-rabbitmq.env`
4. **Delete both temp files** immediately after. The transcript keeps only the commands and paths, never the values.

## 7. Bootstrap capa 2 (guided)
Resolve the image to deploy (latest `v*` tag in the registry) and substitute `__IMAGE__` in each templated manifest, exactly as `bootstrap.ps1` does. Apply the items below **one at a time**: run each, show its output (including the rollout/wait gates), then **report the result and pause for the user's OK before moving to the next** — do this even though these steps aren't destructive, so the user watches progress gate by gate. Order:
1. `namespace.yaml`
2. infra — `infra/postgres.yaml`, `infra/redis.yaml`, `infra/rabbitmq.yaml`; wait `rollout status statefulset/postgres` and `statefulset/rabbitmq`.
3. app — `app/serviceaccount.yaml`, `app/configmap.yaml`; delete + apply `app/migrate-job.yaml` and `app/seed-job.yaml`; `kubectl wait --for=condition=complete` both; then `app/api.yaml`.
4. KEDA — install the operator manifest (`v2.20.1`), wait `deployment/keda-operator` available; then `judge/worker.yaml` and `judge/keda.yaml`.
5. `ingress/ingress.yaml`, `infra/backup-cronjob.yaml`, and `app/cleanup-sessions-cronjob.yaml` (templated — needs `__IMAGE__` resolved same as `api.yaml`/`worker.yaml`).

**Keeping data (from step 0)?** Only when the user chose to preserve or migrate data. Fold these into item 3:
- **Postgres** — the dump from step 1 already holds the full schema and all data (the admin user included). Sequence it carefully: Postgres comes up empty (item 2) → **restore the dump into it** (`psql`/`pg_restore` into the Postgres pod) → run the `migrate` job (it no-ops if the dump is already at the current schema, or applies any newer migrations on top if not) → **skip the `seed` job** (the admin already exists in the dump) → then `api`. Confirm the exact order with the user; restoring at the wrong point collides with the jobs.
- **Uploads** (migration only) — once the new bucket exists (created in step 3), copy the user files over: `gsutil -m rsync -r gs://<old-uploads> gs://<new-uploads>`. A same-project recreate skips this — its bucket survived step 2.

If the user prefers the unattended path here, they can run `bootstrap.ps1` instead — it performs these same steps (secrets via its `secrets.env` template — internals pre-generated, ADMIN and SMTP filled by the user; no data restore).

## 8. DNS + certificate
The Ingress attaches to the named static IP `training-center-ip` (its address is the `ingress_ip` output) and serves `api.trainingjudgecenter.com` over HTTPS via a Google-managed certificate (`api-cert`), with HTTP redirected to HTTPS. The domain's source of truth is the `ManagedCertificate`/`Ingress` manifest.
- Point the **A record for `api.trainingjudgecenter.com`** at `ingress_ip`, at the domain's DNS provider. If the reserved IP survived the teardown its value is unchanged — skip this; if it was recreated or you migrated, it's a new IP that must be updated.
- The managed certificate only provisions **after** DNS resolves to the IP and the load balancer is serving; it goes Provisioning → Active over several minutes (sometimes 15–60). Watch it with `kubectl get managedcertificate api-cert -n training-center`. Until it's Active, HTTPS returns certificate errors — that's expected, not a failure.

## 9. NetworkPolicy (hardening)
Apply `deploy/k8s/network/policies.yaml` (the unattended `bootstrap.ps1` already applies it as its last step). It's an **ingress-only** pass: a `default-deny-ingress` for the namespace, plus allows for the real flows — Google LB ranges → `api:8080`; same-namespace → `postgres:5432`, `redis:6379`, `rabbitmq:5672`; and the `keda` namespace → `rabbitmq:5672`. Egress is left open on purpose (DNS, GCS, SMTP, metadata, the judge's image pulls); locking it down is a deliberate phase 2. These only enforce on a Dataplane V2 cluster. **After applying, verify connectivity** — `kubectl get pods -n training-center` all healthy, KEDA still reads the queue, and the API answers through the Ingress — since DPv2 drops disallowed traffic silently. (Untrusted-code isolation is separate: submission containers already run with no network via `NetworkMode: "none"` in the executor.)

## 10. Verify
Start with the fast gate: `kubectl get pods -n training-center` (all Running/Completed), the API health endpoint through the Ingress once the cert is Active, and admin login with the password set in step 6. If data was kept (step 0), also confirm the preserved data is actually visible through the API (e.g. existing problems/submissions show up) — a successful login alone doesn't prove the restore worked.

That gate doesn't touch the judge — the actual reason this environment exists — so go further:

- **A published problem already exists** (the common case — preserved data, or a same-project recreate that kept the DB): submit a known-good solution — `POST /problems/{slug}/submissions` — then poll `GET /submissions/{submissionId}` until `status` leaves `PENDING`/`RUNNING`. A real verdict (ideally `ACCEPTED`; anything final other than `SYSTEM_ERROR`) confirms the **whole chain** — API → RabbitMQ → KEDA scaling the worker 0→1 → judge-worker → DinD → verdict written back.
- **No published problem exists yet** (fresh environment, nothing to submit to — publishing a problem via API is still a pending endpoint, see CLAUDE.md): verify the pieces individually instead — `judge-worker`'s pod is schedulable on the `judge` node pool and its `dind` init container passes its startup probe; `kubectl get scaledobject judge-worker -n training-center` shows KEDA reading the queue metric; RabbitMQ's queue is reachable (management UI or `rabbitmqctl list_queues`).

Fold in the NetworkPolicy check from step 9 here if not already done — confirm nothing above got silently dropped.
