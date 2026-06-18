package analyzer

import (
	"regexp"
	"strings"
)

// dbClients are command-line database clients. When one of these runs inline
// SQL (psql -c "…", mysql -e "…", sqlite3 db "…"), we inspect the statement.
var dbClients = map[string]bool{
	"psql": true, "mysql": true, "mariadb": true, "mysqlsh": true,
	"sqlite3": true, "mongosh": true, "mongo": true, "cockroach": true,
	"cqlsh": true, "clickhouse-client": true, "usql": true, "pgcli": true,
	"mycli": true, "litecli": true,
}

var (
	reDropDatabase = regexp.MustCompile(`(?i)\bDROP\s+(?:DATABASE|SCHEMA)\s+(?:IF\s+EXISTS\s+)?` + "`?\"?([\\w.]+)")
	reDropTable    = regexp.MustCompile(`(?i)\bDROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?` + "`?\"?([\\w.]+)")
	reTruncate     = regexp.MustCompile(`(?i)\bTRUNCATE\s+(?:TABLE\s+)?` + "`?\"?([\\w.]+)")
	reDeleteFrom   = regexp.MustCompile(`(?i)\bDELETE\s+FROM\s+` + "`?\"?([\\w.]+)")
	reUpdateSet    = regexp.MustCompile(`(?i)\bUPDATE\s+` + "`?\"?([\\w.]+)`?\"?\\s+SET\\b")
	reHasWhere     = regexp.MustCompile(`(?i)\bWHERE\b`)
)

// ruleSQL flags destructive SQL: dropping databases/tables, truncating, and —
// the classic disasters — DELETE or UPDATE with no WHERE clause.
func ruleSQL(c Command) []Finding {
	w := c.Words()
	if len(w) == 0 {
		return nil
	}
	first := strings.ToUpper(w[0])
	isRawSQL := first == "DROP" || first == "TRUNCATE" || first == "DELETE" || first == "UPDATE"
	if !dbClients[c.Name()] && !isRawSQL {
		return nil
	}

	// Evaluate each statement separately so a WHERE on one statement can't mask
	// an unguarded DELETE/UPDATE in another.
	var findings []Finding
	for _, stmt := range strings.Split(strings.Join(w, " "), ";") {
		findings = append(findings, sqlStatement(stmt, c)...)
	}
	return findings
}

func sqlStatement(stmt string, c Command) []Finding {
	hasWhere := reHasWhere.MatchString(stmt)
	switch {
	case reDropDatabase.MatchString(stmt):
		m := reDropDatabase.FindStringSubmatch(stmt)
		return []Finding{sqlFinding("sql-drop-database", Critical, "Drops a database",
			"DROP DATABASE/SCHEMA "+clean(m[1])+" deletes the database and every table in it. There is no undo without a backup.", c, m[1])}
	case reDropTable.MatchString(stmt):
		m := reDropTable.FindStringSubmatch(stmt)
		return []Finding{sqlFinding("sql-drop-table", Danger, "Drops a table",
			"DROP TABLE "+clean(m[1])+" deletes the table and all of its rows permanently.", c, m[1])}
	case reTruncate.MatchString(stmt):
		m := reTruncate.FindStringSubmatch(stmt)
		return []Finding{sqlFinding("sql-truncate", Danger, "Truncates a table",
			"TRUNCATE "+clean(m[1])+" removes every row in one shot and usually cannot be rolled back.", c, m[1])}
	case reDeleteFrom.MatchString(stmt) && !hasWhere:
		m := reDeleteFrom.FindStringSubmatch(stmt)
		return []Finding{sqlFinding("sql-delete-all", Danger, "DELETE with no WHERE",
			"DELETE FROM "+clean(m[1])+" without a WHERE clause removes every row in the table.", c, m[1])}
	case reUpdateSet.MatchString(stmt) && !hasWhere:
		m := reUpdateSet.FindStringSubmatch(stmt)
		return []Finding{sqlFinding("sql-update-all", Danger, "UPDATE with no WHERE",
			"UPDATE "+clean(m[1])+" with no WHERE clause overwrites that column in every row at once.", c, m[1])}
	}
	return nil
}

func sqlFinding(rule string, sev Severity, title, detail string, c Command, target string) Finding {
	return Finding{Rule: rule, Severity: sev, Title: title, Detail: detail, Command: c.Raw, Targets: []string{clean(target)}}
}

func clean(name string) string {
	return strings.Trim(name, "`\"';")
}
