package migrations

import "embed"

// FS embeds all .up.sql and .down.sql migration files
//go:embed *.sql
var FS embed.FS
