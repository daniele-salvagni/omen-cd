<p align="center">
  <img src="docs/omen.png" alt="omen" height="96px">
</p>

# omen

A pull-based GitOps runner. Watches a branch, runs shell commands when files
change.

```
pull --> match --> run --> notify
```

## Install

```sh
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/;s/armv7l/armv7/')
curl -fsSL -o /tmp/omen https://github.com/daniele-salvagni/omen-cd/releases/latest/download/omen-linux-${ARCH}
sudo install -m 755 /tmp/omen /usr/local/bin/omen && rm /tmp/omen
```

From source: `go install github.com/daniele-salvagni/omen-cd/cmd/omen@latest`

## Quickstart

```sh
# 1. Host config
sudo mkdir -p /etc/omen
omen init host | sudo tee /etc/omen/main.yaml
sudo $EDITOR /etc/omen/main.yaml   # set repo and dir

# 2. Sync spec, in your repo root
omen init spec > .omen.yaml
$EDITOR .omen.yaml                 # define rules
git add .omen.yaml && git commit -m 'omen spec' && git push

# 3. Systemd
omen unit service | sudo tee /etc/systemd/system/omen@.service
omen unit timer   | sudo tee /etc/systemd/system/omen@.timer
sudo systemctl enable --now omen@main.timer

# 4. First deploy
sudo omen --config /etc/omen/main.yaml --apply-all
```

Then `git push` handles every deploy. omen is a one-shot binary; any scheduler
works, systemd is the default.

## Configuration

**Host config** at `/etc/omen/<instance>.yaml`:

```yaml
repo: git@github.com:you/infra.git # required
dir: /srv/infra # required
# branch: main                        # default
# source: .omen.yaml                  # in-repo spec path
# ssh_key: /etc/omen/id_ed25519
```

**Sync spec** in the repo (default `.omen.yaml` at the root):

```yaml
notify: |
  curl -fsS -o /dev/null "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
    -d chat_id="${TELEGRAM_CHAT_ID}" \
    -d text="omen ${OMEN_STATUS} ${OMEN_SHORT}"

rules:
  - name: pangolin
    paths: ["stacks/pangolin/**"]
    run: cd stacks/pangolin && docker compose up -d

  - name: cron jobs
    paths: ["cron.d/**"]
    run: install -m 644 cron.d/* /etc/cron.d/
```

- **Rules** fire in order for each changed file matching `paths`.
- **Notify** runs once after a successful deploy. Env: `OMEN_SHA`, `OMEN_SHORT`,
  `OMEN_STATUS`, `OMEN_INSTANCE`.
- **Globs** use [doublestar](https://github.com/bmatcuk/doublestar) (`**`
  crosses directory boundaries).
- **Secrets** live in `/etc/omen/<instance>.env`. Auto-loaded, `chmod 600`,
  never in git.
- **Multi-instance**: `omen@web.timer` reads `/etc/omen/web.yaml`. Enable more
  timers to run multiple repos or hosts on one machine.

## Commands

```
omen [--config PATH] [--env-file PATH] [--dry-run] [--apply-all]
omen init [host|spec]
omen unit [service|timer] [--user NAME]
omen version
```

- `--dry-run` prints the pending diff and matched rules, no execution.
- `--apply-all` forces every rule against HEAD (initial deploy, redeploy).
- `--env-file` overrides the auto-derived env file.
- `--user NAME` on `omen unit service` injects `User=NAME` and `Group=NAME`
  under `[Service]`.

## Operating

```sh
systemctl status omen@main.timer                       # next run, last activation
systemctl start omen@main.service                      # sync now
journalctl -u omen@main.service -f                     # tail live
journalctl -u omen@main.service --since '1 hour ago'   # past runs
journalctl -u omen@main.service --grep failed          # find failures
```

Retention is journald's (`/etc/systemd/journald.conf`).

## Behavior

- **First run with no state** records HEAD and executes no rules. `--apply-all`
  forces a full initial deploy.
- **Fast-forward only.** Divergence from the remote aborts the sync loudly.
- **State advances only on batch success.** A mid-batch failure retries the same
  diff on the next tick. Rules must be idempotent.
- **Notify fires once per run.** `deployed: <names>` on success,
  `failed:
  <rule>` on error. Notify failures are logged, not fatal.
- **One flock per instance** prevents timer and manual runs from colliding.

## License

Apache 2.0.
