# Coolify Deploy — বউ রিমাইন্ডার API

## How it works

Build binary locally → push to git → Coolify pulls & runs it. No build
step runs on the VPS; the Dockerfile just copies the already-compiled
binary into a small Alpine image.

## Steps

### 1. Build
```bash
make build-linux
```
Produces `boureminder-api` (linux/amd64).

### 2. Push to main
```bash
make deploy
```
(Runs `build-linux`, then `git add`/`commit`/`push`. Or do those three
steps by hand if you want a specific commit message.)

### 3. Coolify
- Watches `main` branch of the repo
- Auto-pulls & restarts on new commits

## First-time Coolify setup
| Setting | Value |
|---|---|
| Source | Git (`main` branch) |
| Build method | **Dockerfile** (it only copies the binary in — no compile step) |
| Port | `8080` |
| Health check path | `/healthz` |
| Env vars | Coolify dashboard — see `.env.example` for the full list |

## Notes
- The binary (`boureminder-api`) is tracked in git — don't add it to `.gitignore`.
- Every deploy needs a fresh `make build-linux` first; Coolify never compiles Go itself.
- Migrations (`goose`, `river migrate-up`) don't run automatically on deploy —
  run `make migrate-up` and `make river-migrate` yourself against
  `DATABASE_DIRECT_URL` before or after pushing a schema change.
- Needs `GOOS=linux GOARCH=amd64`; confirm the VPS architecture with `ssh <vps> uname -m`
  if you're ever unsure (`x86_64` = amd64, `aarch64` = arm64).
