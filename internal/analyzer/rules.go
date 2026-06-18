package analyzer

import (
	"fmt"
	"regexp"
	"strings"
)

// commandRules are applied to every simple command in the line.
var commandRules = []commandRule{
	ruleRm,
	ruleGit,
	ruleDD,
	ruleMkfs,
	ruleDiskWipe,
	ruleRedirectDevice,
	ruleChmod,
	ruleChown,
	ruleFind,
	ruleShred,
	ruleDocker,
	ruleKubectl,
	ruleTerraform,
	ruleRedis,
	ruleSQL,
}

// pipelineRules are applied once to the whole command line.
var pipelineRules = []pipelineRule{
	rulePipeToShell,
	ruleForkBomb,
}

// ---------------------------------------------------------------------------
// rm
// ---------------------------------------------------------------------------

func ruleRm(c Command) []Finding {
	if c.Name() != "rm" {
		return nil
	}
	f := parseFlags(c.Words())
	if len(f.pos) == 0 {
		return nil
	}
	recursive := f.anyShort('r', 'R') || f.hasLong("recursive")
	force := f.hasShort('f') || f.hasLong("force")

	// Catastrophic targets get their own, unmissable finding.
	for _, t := range f.pos {
		if k := catastrophicTarget(t); k != "" {
			detail := fmt.Sprintf("This would try to erase %s. There is almost never a reason to do this; it can destroy your account or the whole system.", k)
			if f.hasLong("no-preserve-root") {
				detail += " --no-preserve-root removes the one safety check that normally stops this."
			}
			return []Finding{{
				Rule:     "rm-catastrophic",
				Severity: Critical,
				Title:    "Recursive delete of " + k,
				Detail:   detail,
				Command:  c.Raw,
				Targets:  f.pos,
			}}
		}
	}

	if !recursive && !force {
		return nil // a plain `rm file` is ordinary; don't nag.
	}

	rule, title, sev := "rm-force", "Force-delete", Caution
	if recursive {
		rule, title, sev = "rm-recursive", "Recursive delete", Danger
		if force {
			title = "Recursive force-delete"
		}
	}
	detail := fmt.Sprintf("Permanently removes %s. rm does not use the trash or Recycle Bin, so this cannot be undone.", humanList(f.pos))
	if hasGlob(f.pos) {
		detail += " The wildcard means everything it matches goes too."
	}
	return []Finding{{Rule: rule, Severity: sev, Title: title, Detail: detail, Command: c.Raw, Targets: f.pos}}
}

// ---------------------------------------------------------------------------
// git
// ---------------------------------------------------------------------------

func ruleGit(c Command) []Finding {
	if c.Name() != "git" {
		return nil
	}
	w := c.Words()
	f := parseFlags(w)
	sub := gitSubcommand(w)

	switch sub {
	case "push":
		return gitPush(c, f)
	case "reset":
		return gitReset(c, f)
	case "clean":
		return gitClean(c, f)
	case "checkout", "restore":
		return gitDiscard(c, f, sub)
	case "branch":
		return gitBranch(c, f)
	case "stash":
		return gitStash(c, w)
	case "filter-branch", "filter-repo":
		return []Finding{{
			Rule: "git-filter", Severity: Danger, Command: c.Raw,
			Title:  "Rewrites the entire history",
			Detail: "git " + sub + " rewrites every commit and changes their SHAs. Collaborators must re-clone, and recovery is hard. Make a backup branch first.",
		}}
	}
	return nil
}

// gitSubcommand finds the first positional that is not a value for a global
// option such as -C <path> or -c <key=value>.
func gitSubcommand(w []string) string {
	for i := 1; i < len(w); i++ {
		a := w[i]
		switch {
		case a == "-C" || a == "-c":
			i++ // skip the option's value
		case strings.HasPrefix(a, "-"):
			// other global flags (--no-pager, --git-dir=…) take no separate value
		default:
			return a
		}
	}
	return ""
}

func gitPush(c Command, f flags) []Finding {
	force := f.hasShort('f') || f.hasLong("force")
	lease := f.hasLong("force-with-lease") || f.hasLong("force-if-includes")
	mirror := f.hasLong("mirror")

	switch {
	case mirror:
		return []Finding{{
			Rule: "git-push-mirror", Severity: Danger, Command: c.Raw,
			Title:  "Mirror-push overwrites every remote ref",
			Detail: "--mirror force-updates all branches and tags on the remote and deletes any it doesn't have locally.",
		}}
	case force:
		return []Finding{{
			Rule: "git-push-force", Severity: Danger, Command: c.Raw, Targets: f.pos,
			Title:  "Force-push rewrites remote history",
			Detail: "Overwrites the remote branch with your local version. Commits that are on the remote but not in your local branch are erased for everyone who pulled them.",
		}}
	case lease:
		return []Finding{{
			Rule: "git-push-lease", Severity: Caution, Command: c.Raw, Targets: f.pos,
			Title:  "Force-push with lease",
			Detail: "Safer than a plain --force: it refuses to overwrite the remote if someone has pushed since you last fetched. Still rewrites history if it succeeds.",
		}}
	}
	// Refspec forms with no -f flag: `git push origin +main` forces; `:branch`
	// deletes a remote branch.
	for _, p := range f.pos {
		if len(p) > 1 && p[0] == '+' {
			return []Finding{{
				Rule: "git-push-force", Severity: Danger, Command: c.Raw, Targets: f.pos,
				Title:  "Force-push rewrites remote history",
				Detail: "A refspec beginning with + (" + p + ") forces the push: it overwrites the remote branch and erases any commits there that you don't have.",
			}}
		}
		if len(p) > 1 && p[0] == ':' {
			return []Finding{{
				Rule: "git-push-delete", Severity: Caution, Command: c.Raw, Targets: f.pos,
				Title:  "Deletes a remote branch",
				Detail: "Pushing the refspec " + p + " deletes that branch on the remote for everyone.",
			}}
		}
	}
	return nil
}

func gitReset(c Command, f flags) []Finding {
	if !f.hasLong("hard") {
		return nil
	}
	return []Finding{{
		Rule: "git-reset-hard", Severity: Danger, Command: c.Raw,
		Title:  "Hard reset discards uncommitted work",
		Detail: "git reset --hard throws away every uncommitted change in your tracked files — they revert to the target commit and are not recoverable through git.",
	}}
}

func gitClean(c Command, f flags) []Finding {
	force := f.hasShort('f') || f.hasLong("force")
	dryRun := f.hasShort('n') || f.hasLong("dry-run")
	if !force || dryRun {
		return nil
	}
	dirs := f.hasShort('d')
	ignored := f.hasShort('x') || f.hasShort('X')
	what := "untracked files"
	if dirs {
		what += " and directories"
	}
	detail := "git clean permanently deletes " + what + " that are not in git — there is no commit to recover them from."
	if ignored {
		detail += " -x also removes ignored files, which often means build output and local secrets like .env."
	}
	return []Finding{{
		Rule: "git-clean", Severity: Danger, Command: c.Raw,
		Title: "git clean removes untracked files", Detail: detail,
	}}
}

func gitDiscard(c Command, f flags, sub string) []Finding {
	// Only the path-discarding forms lose work. `git checkout branch` is fine.
	all := contains(f.pos, ".") || (sub == "restore" && len(f.pos) > 0 && !f.hasLong("staged")) ||
		f.hasShort('f') || f.hasLong("force")
	hasPathSep := strings.Contains(c.Raw, " -- ")
	if !all && !hasPathSep {
		return nil
	}
	sev, scope := Caution, "the files you name"
	if contains(f.pos, ".") || f.hasShort('f') || f.hasLong("force") {
		sev, scope = Danger, "every tracked file"
	}
	return []Finding{{
		Rule: "git-discard", Severity: sev, Command: c.Raw, Targets: f.pos,
		Title:  "Discards local changes",
		Detail: fmt.Sprintf("git %s here overwrites %s with the committed version, throwing away your uncommitted edits.", sub, scope),
	}}
}

func gitBranch(c Command, f flags) []Finding {
	forceDelete := f.hasShort('D') || (f.hasShort('d') && f.hasShort('f'))
	if !forceDelete {
		return nil
	}
	return []Finding{{
		Rule: "git-branch-delete", Severity: Caution, Command: c.Raw, Targets: f.pos,
		Title:  "Force-deletes a branch",
		Detail: "git branch -D deletes the branch even if it isn't merged. Commits that exist only on it can become hard to find (recoverable via the reflog for a while).",
	}}
}

func gitStash(c Command, w []string) []Finding {
	if !contains(w, "clear") && !contains(w, "drop") {
		return nil
	}
	return []Finding{{
		Rule: "git-stash-drop", Severity: Caution, Command: c.Raw,
		Title:  "Discards stashed changes",
		Detail: "git stash clear/drop removes saved stashes. The changes inside them are no longer easy to recover.",
	}}
}

// ---------------------------------------------------------------------------
// dd, mkfs, redirection to a device
// ---------------------------------------------------------------------------

func ruleDD(c Command) []Finding {
	if c.Name() != "dd" {
		return nil
	}
	of := ""
	for _, a := range c.Words()[1:] {
		if strings.HasPrefix(a, "of=") {
			of = strings.TrimPrefix(a, "of=")
		}
	}
	if !isBlockDevice(of) {
		return nil // writing dd output to a regular file (e.g. an image) is routine.
	}
	return []Finding{{
		Rule: "dd-device", Severity: Critical, Command: c.Raw, Targets: []string{of},
		Title:  "dd writes straight to a raw disk",
		Detail: "Writing to " + of + " overwrites the disk's partition table and every byte of data on it. Picking the wrong of= here is the classic way to wipe the wrong drive.",
	}}
}

func ruleMkfs(c Command) []Finding {
	name := c.Name()
	if !strings.HasPrefix(name, "mkfs") && name != "mke2fs" && name != "format" {
		return nil
	}
	var devs []string
	for _, a := range c.Words()[1:] {
		if isBlockDevice(a) {
			devs = append(devs, a)
		}
	}
	if len(devs) == 0 {
		return nil // formatting a regular file image (or bare usage) isn't the hazard.
	}
	return []Finding{{
		Rule: "mkfs", Severity: Critical, Command: c.Raw, Targets: devs,
		Title:  "Formats a filesystem",
		Detail: "Creating a new filesystem on " + strings.Join(devs, ", ") + " erases everything currently stored there.",
	}}
}

func ruleDiskWipe(c Command) []Finding {
	var tool string
	switch c.Name() {
	case "wipefs":
		tool = "wipefs"
	case "blkdiscard":
		tool = "blkdiscard"
	default:
		return nil
	}
	var devs []string
	for _, a := range c.Words()[1:] {
		if isBlockDevice(a) {
			devs = append(devs, a)
		}
	}
	if len(devs) == 0 {
		return nil
	}
	return []Finding{{
		Rule: "disk-wipe", Severity: Critical, Command: c.Raw, Targets: devs,
		Title:  tool + " erases a disk",
		Detail: tool + " wipes the filesystem signatures or discards every block on " + strings.Join(devs, ", ") + ", leaving the data unrecoverable.",
	}}
}

func ruleRedirectDevice(c Command) []Finding {
	w := c.Args
	for i := 0; i < len(w)-1; i++ {
		if (w[i] == ">" || w[i] == ">>") && isBlockDevice(w[i+1]) {
			return []Finding{{
				Rule: "redirect-device", Severity: Critical, Command: c.Raw, Targets: []string{w[i+1]},
				Title:  "Redirects output onto a raw disk",
				Detail: "Sending output to " + w[i+1] + " writes over the raw device and corrupts whatever filesystem lives there.",
			}}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// chmod / chown
// ---------------------------------------------------------------------------

func ruleChmod(c Command) []Finding {
	if c.Name() != "chmod" {
		return nil
	}
	f := parseFlags(c.Words())
	recursive := f.hasShort('R') || f.hasLong("recursive")
	if len(f.pos) == 0 {
		return nil
	}
	mode := f.pos[0]
	targets := f.pos[1:]

	for _, t := range targets {
		if catastrophicTarget(t) != "" || isSystemDir(t) {
			return []Finding{{
				Rule: "chmod-system", Severity: Danger, Command: c.Raw, Targets: targets,
				Title:  "Changes permissions on a system path",
				Detail: "Re-permissioning " + t + " can break tools that rely on specific ownership and modes, and may make the system insecure or unbootable.",
			}}
		}
	}
	worldWritable := mode == "777" || mode == "0777" || strings.Contains(mode, "o+w") || strings.Contains(mode, "a+rwx")
	if recursive && worldWritable {
		return []Finding{{
			Rule: "chmod-777", Severity: Danger, Command: c.Raw, Targets: targets,
			Title:  "Makes a tree world-writable",
			Detail: "chmod -R 777 gives every user read, write, and execute on everything under " + humanList(targets) + ". It's a common security mistake and hard to walk back precisely.",
		}}
	}
	return nil
}

func ruleChown(c Command) []Finding {
	if c.Name() != "chown" {
		return nil
	}
	f := parseFlags(c.Words())
	recursive := f.hasShort('R') || f.hasLong("recursive")
	if !recursive || len(f.pos) < 2 {
		return nil
	}
	for _, t := range f.pos[1:] {
		if catastrophicTarget(t) != "" || isSystemDir(t) {
			return []Finding{{
				Rule: "chown-system", Severity: Danger, Command: c.Raw, Targets: f.pos[1:],
				Title:  "Recursively changes ownership of a system path",
				Detail: "chown -R on " + t + " can lock the system or its services out of files they need, and is tedious to reverse.",
			}}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// find -delete / -exec rm
// ---------------------------------------------------------------------------

func ruleFind(c Command) []Finding {
	if c.Name() != "find" {
		return nil
	}
	w := c.Words()
	deletes := contains(w, "-delete")
	execRm := false
	for i, a := range w {
		if (a == "-exec" || a == "-execdir") && i+1 < len(w) && base(w[i+1]) == "rm" {
			execRm = true
		}
	}
	if !deletes && !execRm {
		return nil
	}
	how := "-delete"
	if execRm {
		how = "-exec rm"
	}
	return []Finding{{
		Rule: "find-delete", Severity: Danger, Command: c.Raw,
		Title:  "find deletes every match",
		Detail: "find … " + how + " removes every file that matches the search. If the filter is broader than you think, it takes more with it — and nothing goes to the trash.",
	}}
}

func ruleShred(c Command) []Finding {
	if c.Name() != "shred" {
		return nil
	}
	f := parseFlags(c.Words())
	if len(f.pos) == 0 {
		return nil
	}
	return []Finding{{
		Rule: "shred", Severity: Danger, Command: c.Raw, Targets: f.pos,
		Title:  "Securely erases files",
		Detail: "shred overwrites " + humanList(f.pos) + " so the contents cannot be recovered by any tool. There is no undo.",
	}}
}

// ---------------------------------------------------------------------------
// container / infra ops
// ---------------------------------------------------------------------------

func ruleDocker(c Command) []Finding {
	if n := c.Name(); n != "docker" && n != "podman" {
		return nil
	}
	w := c.Words()
	f := parseFlags(w)
	switch {
	case contains(w, "volume") && (contains(w, "rm") || contains(w, "prune")):
		return []Finding{{
			Rule: "docker-volume", Severity: Danger, Command: c.Raw,
			Title:  "Deletes Docker volumes",
			Detail: "Removing volumes deletes the data inside them — databases, uploads, anything persisted outside the container image is gone.",
		}}
	case contains(w, "prune"):
		sev := Caution
		detail := "docker prune removes stopped containers, unused networks and dangling images to reclaim space."
		if f.hasShort('a') || f.hasLong("all") || f.hasLong("volumes") {
			sev = Danger
			detail = "With --all/--volumes this also removes images you may need to rebuild and volumes holding real data."
		}
		return []Finding{{Rule: "docker-prune", Severity: sev, Command: c.Raw, Title: "Prunes Docker resources", Detail: detail}}
	}
	return nil
}

func ruleKubectl(c Command) []Finding {
	if c.Name() != "kubectl" && c.Name() != "oc" {
		return nil
	}
	w := c.Words()
	if !contains(w, "delete") {
		return nil
	}
	f := parseFlags(w)
	sev, detail := Caution, "kubectl delete removes the named resources from the cluster."
	if f.hasLong("all") || contains(w, "namespace") || contains(w, "ns") {
		sev = Danger
		detail = "Deleting a namespace or using --all tears down many live resources at once — running workloads and their data go with them."
	}
	return []Finding{{Rule: "kubectl-delete", Severity: sev, Command: c.Raw, Title: "Deletes cluster resources", Detail: detail}}
}

func ruleTerraform(c Command) []Finding {
	if n := c.Name(); n != "terraform" && n != "tofu" {
		return nil
	}
	w := c.Words()
	f := parseFlags(w)
	switch {
	case contains(w, "destroy"):
		return []Finding{{
			Rule: "terraform-destroy", Severity: Danger, Command: c.Raw,
			Title:  "Destroys managed infrastructure",
			Detail: "terraform destroy tears down every resource in this state — servers, databases, buckets. Rebuilding does not bring back the data they held.",
		}}
	case contains(w, "apply") && (f.hasLong("auto-approve") || contains(w, "-auto-approve")):
		return []Finding{{
			Rule: "terraform-apply-auto", Severity: Caution, Command: c.Raw,
			Title:  "Applies infra changes without review",
			Detail: "-auto-approve skips the plan confirmation, so any replacements or deletions in the plan happen immediately.",
		}}
	}
	return nil
}

func ruleRedis(c Command) []Finding {
	if c.Name() != "redis-cli" {
		return nil
	}
	w := strings.ToUpper(strings.Join(c.Words(), " "))
	if strings.Contains(w, "FLUSHALL") || strings.Contains(w, "FLUSHDB") {
		return []Finding{{
			Rule: "redis-flush", Severity: Danger, Command: c.Raw,
			Title:  "Flushes the Redis database",
			Detail: "FLUSHALL/FLUSHDB deletes every key. If persistence is off, the data is simply gone.",
		}}
	}
	return nil
}

// ---------------------------------------------------------------------------
// pipeline-level rules
// ---------------------------------------------------------------------------

var downloaders = map[string]bool{"curl": true, "wget": true, "fetch": true, "http": true, "httpie": true}
var interpreters = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "dash": true, "ksh": true, "fish": true,
	"python": true, "python3": true, "perl": true, "ruby": true, "node": true, "php": true,
}

func rulePipeToShell(pl Pipeline) []Finding {
	if !pl.Piped || len(pl.Commands) < 2 {
		return nil
	}
	sawDownload := false
	for _, c := range pl.Commands {
		n := c.Name()
		if downloaders[n] {
			sawDownload = true
			continue
		}
		if sawDownload && interpreters[n] {
			return []Finding{{
				Rule: "pipe-to-shell", Severity: Danger, Command: pl.Commands[0].Raw + " | " + c.Raw,
				Title:  "Runs a downloaded script unread",
				Detail: "Piping " + n + " straight from a download executes whatever the server sends, with your privileges, before you ever see it. Download first, read it, then run it.",
			}}
		}
	}
	return nil
}

var forkBomb = regexp.MustCompile(`(\w+|:)\s*\(\s*\)\s*\{[^}]*[|&]\s*(\w+|:)[^}]*&[^}]*\}`)

func ruleForkBomb(pl Pipeline) []Finding {
	line := pl.input()
	if strings.Contains(strings.ReplaceAll(line, " ", ""), ":(){:|:&};:") || forkBomb.MatchString(line) {
		return []Finding{{
			Rule: "fork-bomb", Severity: Critical, Command: strings.TrimSpace(line),
			Title:  "Fork bomb",
			Detail: "This defines a function that calls itself twice forever, spawning processes until the machine runs out of resources and locks up.",
		}}
	}
	return nil
}

func (pl Pipeline) input() string {
	if pl.Raw != "" {
		return pl.Raw
	}
	parts := make([]string, len(pl.Commands))
	for i, c := range pl.Commands {
		parts[i] = c.Raw
	}
	return strings.Join(parts, " ; ")
}
