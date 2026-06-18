package analyzer

// RuleInfo describes one detector for the `oops-guard rules` listing and docs.
type RuleInfo struct {
	ID       string   `json:"id"`
	Severity Severity `json:"severity"`
	Summary  string   `json:"summary"`
}

// Catalog returns every rule oops-guard ships with, grouped roughly by topic
// and ordered from most to least severe within each group. It is the single
// source of truth behind `oops-guard rules` and docs/rules.md.
func Catalog() []RuleInfo {
	return []RuleInfo{
		// Filesystem
		{"rm-catastrophic", Critical, "rm -rf of /, ~, $HOME, or a top-level system directory"},
		{"rm-recursive", Danger, "rm -r / -rf of a directory tree (with a precise file + size preview)"},
		{"rm-force", Caution, "rm -f of specific files (skips the trash)"},
		{"find-delete", Danger, "find … -delete or -exec rm removes every match"},
		{"shred", Danger, "shred irreversibly overwrites files"},
		// Disks & devices
		{"dd-device", Critical, "dd of=/dev/… overwrites a raw disk"},
		{"mkfs", Critical, "mkfs.* formats a device, erasing it"},
		{"disk-wipe", Critical, "wipefs / blkdiscard erases a disk's signatures or blocks"},
		{"redirect-device", Critical, "redirecting output (>) onto a raw disk device"},
		// Permissions
		{"chmod-777", Danger, "chmod -R 777 makes a tree world-writable"},
		{"chmod-system", Danger, "chmod on /etc, /usr, and other system paths"},
		{"chown-system", Danger, "chown -R on a system path"},
		// git
		{"git-push-force", Danger, "git push --force (or a +refspec) rewrites remote history"},
		{"git-push-mirror", Danger, "git push --mirror overwrites every remote ref"},
		{"git-push-delete", Caution, "git push origin :branch deletes a remote branch"},
		{"git-push-lease", Caution, "git push --force-with-lease (the safer force-push)"},
		{"git-reset-hard", Danger, "git reset --hard discards uncommitted work"},
		{"git-clean", Danger, "git clean -f[dx] deletes untracked (and ignored) files"},
		{"git-discard", Danger, "git checkout/restore that throws away local changes"},
		{"git-branch-delete", Caution, "git branch -D force-deletes a branch"},
		{"git-stash-drop", Caution, "git stash clear/drop discards stashes"},
		{"git-filter", Danger, "git filter-branch/filter-repo rewrites all history"},
		// Databases
		{"sql-drop-database", Critical, "DROP DATABASE / DROP SCHEMA"},
		{"sql-drop-table", Danger, "DROP TABLE"},
		{"sql-truncate", Danger, "TRUNCATE removes every row"},
		{"sql-delete-all", Danger, "DELETE with no WHERE clause"},
		{"sql-update-all", Danger, "UPDATE with no WHERE clause"},
		{"redis-flush", Danger, "redis-cli FLUSHALL / FLUSHDB"},
		// Infra & ops
		{"terraform-destroy", Danger, "terraform/tofu destroy tears down infrastructure"},
		{"terraform-apply-auto", Caution, "terraform apply -auto-approve skips the plan review"},
		{"docker-volume", Danger, "docker/podman volume rm or prune deletes volume data"},
		{"docker-prune", Caution, "docker system prune (escalates with --all/--volumes)"},
		{"kubectl-delete", Danger, "kubectl delete --all or of a namespace"},
		// Shell hazards
		{"pipe-to-shell", Danger, "curl | sh — running a downloaded script unread"},
		{"fork-bomb", Critical, "a self-replicating fork bomb"},
	}
}
