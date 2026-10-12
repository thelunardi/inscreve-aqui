// Package migrate aplica as migrations do pacote migrations com o goose (ADR-0010).
// É usado pelo cmd/migrate e pelos testes, para que os dois apliquem o schema
// exatamente da mesma forma.
package migrate

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // driver "pgx" para database/sql, exigido pelo goose
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/thelunardi/inscreve-aqui/migrations"
)

// Migrator aplica e desfaz migrations em um banco.
type Migrator struct {
	db       *sql.DB
	provider *goose.Provider
}

// Open conecta ao banco de url. Feche com Close.
//
// Cada migration roda em uma transação (padrão do goose), e um advisory lock
// impede que duas execuções apliquem migrations ao mesmo tempo.
func Open(url string) (*Migrator, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create session locker: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS, goose.WithSessionLocker(locker))
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create provider: %w", err)
	}
	return &Migrator{db: db, provider: provider}, nil
}

// Close fecha a conexão com o banco.
func (m *Migrator) Close() error {
	return m.provider.Close()
}

// Up aplica todas as migrations pendentes.
func (m *Migrator) Up(ctx context.Context) ([]*goose.MigrationResult, error) {
	return m.provider.Up(ctx)
}

// Down desfaz a última migration aplicada.
func (m *Migrator) Down(ctx context.Context) (*goose.MigrationResult, error) {
	return m.provider.Down(ctx)
}

// DownAll desfaz todas as migrations aplicadas.
func (m *Migrator) DownAll(ctx context.Context) ([]*goose.MigrationResult, error) {
	return m.provider.DownTo(ctx, 0)
}

// Status lista cada migration com seu estado (aplicada ou pendente).
func (m *Migrator) Status(ctx context.Context) ([]*goose.MigrationStatus, error) {
	return m.provider.Status(ctx)
}
