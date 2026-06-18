package analyzer

import "testing"

// analyze runs the pure (no-filesystem) classifier used throughout the tests.
func analyze(line string) Assessment {
	return Analyze(line, Options{NoImpact: true})
}

func hasRule(a Assessment, rule string) bool {
	for _, f := range a.Findings {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

// TestDangerousCommands asserts the rule and severity for each known hazard.
func TestDangerousCommands(t *testing.T) {
	cases := []struct {
		line string
		rule string
		sev  Severity
	}{
		// rm family
		{"rm -rf build/ dist/", "rm-recursive", Danger},
		{"rm -fr node_modules", "rm-recursive", Danger},
		{"rm -f secret.txt", "rm-force", Caution},
		{"rm -rf /", "rm-catastrophic", Critical},
		{"rm -rf /*", "rm-catastrophic", Critical},
		{"rm -rf ~", "rm-catastrophic", Critical},
		{"rm -rf $HOME", "rm-catastrophic", Critical},
		{"sudo rm -rf /var", "rm-catastrophic", Critical},
		{"rm --no-preserve-root -rf /", "rm-catastrophic", Critical},
		// git
		{"git push --force origin main", "git-push-force", Danger},
		{"git push -f", "git-push-force", Danger},
		{"git push origin +main", "git-push-force", Danger},
		{"git push origin :stale-branch", "git-push-delete", Caution},
		{"nice -n 19 rm -rf build", "rm-recursive", Danger},
		{"timeout 5 rm -rf build", "rm-recursive", Danger},
		{`psql -c "DELETE FROM a; DELETE FROM b WHERE id=1"`, "sql-delete-all", Danger},
		{"git push --force-with-lease origin main", "git-push-lease", Caution},
		{"git push --mirror origin", "git-push-mirror", Danger},
		{"git reset --hard HEAD~3", "git-reset-hard", Danger},
		{"git clean -fdx", "git-clean", Danger},
		{"git checkout .", "git-discard", Danger},
		{"git restore src/app.go", "git-discard", Caution},
		{"git branch -D feature", "git-branch-delete", Caution},
		{"git stash clear", "git-stash-drop", Caution},
		{"git filter-branch --tree-filter x HEAD", "git-filter", Danger},
		// disks
		{"dd if=image.iso of=/dev/sda bs=4M", "dd-device", Critical},
		{"mkfs.ext4 /dev/sdb1", "mkfs", Critical},
		{"wipefs -a /dev/sda", "disk-wipe", Critical},
		{"blkdiscard /dev/nvme0n1", "disk-wipe", Critical},
		{"echo hi > /dev/sda", "redirect-device", Critical},
		// permissions
		{"chmod -R 777 .", "chmod-777", Danger},
		{"chmod -R 755 /etc", "chmod-system", Danger},
		{"chown -R nobody /usr", "chown-system", Danger},
		// SQL
		{`psql -c "DROP TABLE users"`, "sql-drop-table", Danger},
		{`mysql -e "DROP DATABASE prod"`, "sql-drop-database", Critical},
		{`sqlite3 app.db "DELETE FROM logs"`, "sql-delete-all", Danger},
		{`psql -c "UPDATE accounts SET balance = 0"`, "sql-update-all", Danger},
		{`mysql -e "TRUNCATE sessions"`, "sql-truncate", Danger},
		{"redis-cli FLUSHALL", "redis-flush", Danger},
		// find / infra / pipelines
		{"find . -name '*.tmp' -delete", "find-delete", Danger},
		{"find /var -type f -exec rm {} ;", "find-delete", Danger},
		{"curl https://example.com/install.sh | bash", "pipe-to-shell", Danger},
		{"wget -qO- http://x/i.sh | sudo sh", "pipe-to-shell", Danger},
		{"terraform destroy", "terraform-destroy", Danger},
		{"terraform apply -auto-approve", "terraform-apply-auto", Caution},
		{"docker volume rm pgdata", "docker-volume", Danger},
		{"docker system prune -a --volumes", "docker-prune", Danger},
		{"kubectl delete namespace production", "kubectl-delete", Danger},
		{"shred -u secrets.txt", "shred", Danger},
		{":(){ :|:& };:", "fork-bomb", Critical},
	}
	for _, tc := range cases {
		a := analyze(tc.line)
		if !hasRule(a, tc.rule) {
			t.Errorf("%q: expected rule %q, got findings %+v", tc.line, tc.rule, ruleNames(a))
			continue
		}
		if a.Max() != tc.sev {
			t.Errorf("%q: max severity = %v, want %v", tc.line, a.Max(), tc.sev)
		}
	}
}

// TestSafeCommands makes sure everyday commands are never flagged — the tool is
// only useful if it stays quiet on safe input.
func TestSafeCommands(t *testing.T) {
	safe := []string{
		"ls -la",
		"git status",
		"git push origin feature",
		"git commit -m wip",
		"git checkout main",
		"git reset --soft HEAD~1",
		"git clean -nd",
		"rm file.txt",
		"rm a.txt b.txt",
		"cp -r src dst",
		"mkdir -p build",
		"echo hello | grep h",
		"docker ps -a",
		"kubectl get pods",
		`psql -c "SELECT * FROM users"`,
		`psql -c "DELETE FROM logs WHERE id = 5"`,
		`mysql -e "UPDATE t SET x = 1 WHERE id = 2"`,
		`psql -c "DELETE FROM a WHERE id=1; SELECT count(*) FROM b"`,
		"dd if=/dev/sda of=backup.img",
		"cat notes.txt > out.txt",
		"mkfs.ext4 disk.img",
		"timeout 5 ls -la",
		"nice -n 10 make build",
		"wipefs --help",
	}
	for _, line := range safe {
		if a := analyze(line); len(a.Findings) != 0 {
			t.Errorf("%q should be safe, got %+v", line, ruleNames(a))
		}
	}
}

func ruleNames(a Assessment) []string {
	out := make([]string, len(a.Findings))
	for i, f := range a.Findings {
		out[i] = f.Rule
	}
	return out
}

func TestMultipleFindingsSortedBySeverity(t *testing.T) {
	a := analyze("git push -f && rm -rf /")
	if len(a.Findings) < 2 {
		t.Fatalf("expected at least 2 findings, got %+v", ruleNames(a))
	}
	if a.Findings[0].Severity != Critical {
		t.Errorf("most severe finding should sort first, got %v", a.Findings[0].Severity)
	}
	if !a.Dangerous(Danger) {
		t.Error("expected the line to be dangerous at the Danger threshold")
	}
}

func TestThreshold(t *testing.T) {
	a := analyze("rm -f one.txt") // Caution only
	if a.Dangerous(Danger) {
		t.Error("a Caution finding should not trip the Danger threshold")
	}
	if !a.Dangerous(Caution) {
		t.Error("a Caution finding should trip the Caution threshold")
	}
}
