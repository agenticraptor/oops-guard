# Detection rules

oops-guard ships with a deterministic catalog of rules. They run with no network
and no model — fast enough to sit in front of every command you type. The
guiding principle is **specificity**: each rule targets a genuinely destructive
pattern and is paired with tests proving it stays quiet on the everyday commands
that look similar.

Run `oops-guard rules` for this list at any time (add `--json` for tooling).

## Severity

| Severity | Meaning | Default behavior in the shell |
|----------|---------|-------------------------------|
| **caution** | Recoverable or scoped loss worth a glance | Not interrupted (below threshold) |
| **danger** | Irreversible loss of local work or data | **Prompts** (y/N) |
| **critical** | Catastrophic, machine- or database-wide | **Prompts** (type-to-confirm) |

The threshold that interrupts your shell is configurable; see
[configuration.md](configuration.md).

## The catalog

```text
CRITICAL  rm-catastrophic        rm -rf of /, ~, $HOME, or a top-level system directory
DANGER    rm-recursive           rm -r / -rf of a directory tree (with a precise file + size preview)
CAUTION   rm-force               rm -f of specific files (skips the trash)
DANGER    find-delete            find … -delete or -exec rm removes every match
DANGER    shred                  shred irreversibly overwrites files
CRITICAL  dd-device              dd of=/dev/… overwrites a raw disk
CRITICAL  mkfs                   mkfs.* formats a device, erasing it
CRITICAL  disk-wipe              wipefs / blkdiscard erases a disk's signatures or blocks
CRITICAL  redirect-device        redirecting output (>) onto a raw disk device
DANGER    chmod-777              chmod -R 777 makes a tree world-writable
DANGER    chmod-system           chmod on /etc, /usr, and other system paths
DANGER    chown-system           chown -R on a system path
DANGER    git-push-force         git push --force (or a +refspec) rewrites remote history
DANGER    git-push-mirror        git push --mirror overwrites every remote ref
CAUTION   git-push-delete        git push origin :branch deletes a remote branch
CAUTION   git-push-lease         git push --force-with-lease (the safer force-push)
DANGER    git-reset-hard         git reset --hard discards uncommitted work
DANGER    git-clean              git clean -f[dx] deletes untracked (and ignored) files
DANGER    git-discard            git checkout/restore that throws away local changes
CAUTION   git-branch-delete      git branch -D force-deletes a branch
CAUTION   git-stash-drop         git stash clear/drop discards stashes
DANGER    git-filter             git filter-branch/filter-repo rewrites all history
CRITICAL  sql-drop-database      DROP DATABASE / DROP SCHEMA
DANGER    sql-drop-table         DROP TABLE
DANGER    sql-truncate           TRUNCATE removes every row
DANGER    sql-delete-all         DELETE with no WHERE clause
DANGER    sql-update-all         UPDATE with no WHERE clause
DANGER    redis-flush            redis-cli FLUSHALL / FLUSHDB
DANGER    terraform-destroy      terraform/tofu destroy tears down infrastructure
CAUTION   terraform-apply-auto   terraform apply -auto-approve skips the plan review
DANGER    docker-volume          docker/podman volume rm or prune deletes volume data
CAUTION   docker-prune           docker system prune (escalates with --all/--volumes)
DANGER    kubectl-delete         kubectl delete --all or of a namespace
DANGER    pipe-to-shell          curl | sh — running a downloaded script unread
CRITICAL  fork-bomb              a self-replicating fork bomb
```

## What it deliberately ignores

To avoid becoming a nag, oops-guard says nothing about, for example:

- `rm file.txt` (a single, non-recursive file removal)
- `git push origin feature`, `git reset --soft`, `git clean -nd` (dry run)
- `git checkout main` (switching branches — no local changes discarded)
- `dd … of=disk.img` (writing to a regular file, e.g. creating an image)
- `DELETE FROM t WHERE id = 5`, `UPDATE t SET x = 1 WHERE …` (scoped statements)
- `docker ps`, `kubectl get`, `terraform plan`

If you find a false positive (a safe command that gets flagged) or a false
negative (a dangerous command that slips through), please
[open an issue](https://github.com/agenticraptor/oops-guard/issues/new/choose) —
both are bugs worth fixing.

## The impact preview

For destructive filesystem and git commands, oops-guard adds a **read-only**
preview so the warning is concrete rather than generic:

- **Files & size:** a bounded walk reports how many files and how many bytes a
  delete would remove (shown as `≥ N` if it hits the safety limit).
- **git state:** if a target is inside a git repo, it reports the number of
  uncommitted changes that would go with it.
- **Notes:** `outside the current directory`, `symlink — removes the link, not
  its target`, `wildcard`, and `not found — nothing to delete here`.

The walk is capped (20,000 files / 400 ms) so it never stalls your prompt, and
catastrophic targets like `/` are never walked.
