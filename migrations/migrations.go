// Package migrations embute os arquivos SQL do schema (ADR-0010).
// Os mesmos arquivos são lidos pelo sqlc, pelos testes e pelo cmd/migrate.
package migrations

import "embed"

// FS contém todas as migrations, em ordem pelo número do arquivo.
//
//go:embed *.sql
var FS embed.FS
