module github.com/UNO-SOFT/mantisbt-plugins-Jira

go 1.23.0

toolchain go1.24.1

require (
	github.com/UNO-SOFT/zlog v0.8.5
	github.com/google/renameio/v2 v2.0.0
	github.com/klauspost/compress v1.18.0
	github.com/oklog/ulid/v2 v2.1.0
	github.com/peterbourgon/ff/v4 v4.0.0-alpha.4
	github.com/rjeczalik/notify v0.9.3
	github.com/rogpeppe/retry v0.1.0
	github.com/tgulacsi/go v0.27.8
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	golang.org/x/exp v0.0.0-20250305212735-054e65f0b394 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/term v0.30.0 // indirect
)

replace github.com/UNO-SOFT/zlog => ../zlog
