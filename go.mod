module github.com/UNO-SOFT/mantisbt-plugins-Jira

go 1.26.0

require (
	github.com/UNO-SOFT/zlog v0.8.6
	github.com/google/renameio/v2 v2.0.2
	github.com/klauspost/compress v1.20.0
	github.com/oklog/ulid/v2 v2.1.2
	github.com/peterbourgon/ff/v4 v4.0.0-beta.1
	github.com/rjeczalik/notify v0.9.3
	github.com/rogpeppe/retry v0.1.0
	github.com/tgulacsi/go v0.30.0
	golang.org/x/time v0.16.0
)

require (
	github.com/go-logr/logr v1.4.4 // indirect
	golang.org/x/exp v0.0.0-20260908205506-85c1c2202aba // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/term v0.46.0 // indirect
)

// replace github.com/UNO-SOFT/zlog => ../zlog

replace github.com/peterbourgon/ff/v4 v4.0.0-beta.1 => github.com/UNO-SOFT/ff/v4 v4.0.0-beta.1.us
