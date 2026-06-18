package analyzer

import "strings"

// Severity ranks how dangerous a command is. Higher is worse.
type Severity int

// Severity levels, from harmless to catastrophic.
const (
	// Safe means no destructive behavior was detected.
	Safe Severity = iota
	// Caution is a reversible or recoverable change worth a glance.
	Caution
	// Danger is irreversible loss of local work or data if you proceed.
	Danger
	// Critical is catastrophic, machine- or database-wide destruction.
	Critical
)

// String returns the lowercase name of the severity.
func (s Severity) String() string {
	switch s {
	case Critical:
		return "critical"
	case Danger:
		return "danger"
	case Caution:
		return "caution"
	default:
		return "safe"
	}
}

// Label returns the short upper-case tag shown in reports.
func (s Severity) Label() string {
	return strings.ToUpper(s.String())
}

// MarshalJSON renders the severity as its name so JSON output is readable.
func (s Severity) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// ParseSeverity converts a name ("caution", "danger", "critical", "safe") into
// a Severity. Unknown names fall back to Danger, the safest useful default.
func ParseSeverity(name string) Severity {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "safe", "info", "none":
		return Safe
	case "caution", "warn", "warning", "low":
		return Caution
	case "danger", "high":
		return Danger
	case "critical", "catastrophic", "max":
		return Critical
	default:
		return Danger
	}
}
