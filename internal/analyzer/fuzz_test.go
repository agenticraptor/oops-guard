package analyzer

import "testing"

// FuzzAnalyze asserts the safety-critical invariant that the classifier is a
// total function: it must never panic, no matter how malformed the command
// line is. oops-guard runs on every command in a hooked shell, so a panic here
// would be a denial of service against the user's own terminal.
//
// Running `go test` exercises the seed corpus below; `go test -fuzz=FuzzAnalyze`
// explores further.
func FuzzAnalyze(f *testing.F) {
	seeds := []string{
		"",
		"   ",
		"rm -rf /",
		"git push --force origin main",
		`psql -c "DROP TABLE users"`,
		":(){ :|:& };:",
		"a && b | c ; d &",
		`rm -rf "a b" 'c d'`,
		"sudo  rm   -rf   $HOME",
		"find . -name '*.tmp' -delete",
		`"`,
		"'",
		"\\",
		"|||",
		"&&&&",
		">>> < >",
		"rm -rf ~/",
		"nice -n 19 timeout 5 sudo rm -rf /",
		"dd if=/dev/zero of=",
		"chmod -R",
		"git",
		"--",
		"-",
		string([]byte{0x00, 0x01, 0x02}),
		"rm -rf 𝓊𝓃𝒾𝒸ℴ𝒹ℯ/💀",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, line string) {
		pl := Parse(line)
		_ = pl.input()
		for _, c := range pl.Commands {
			_ = c.Name()
			_ = c.Words()
		}
		// Pure analysis (no filesystem) must also never panic.
		a := Analyze(line, Options{NoImpact: true})
		_ = a.Max()
		_ = a.Dangerous(Danger)
	})
}
