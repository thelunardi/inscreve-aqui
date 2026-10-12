package migrate_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/thelunardi/inscreve-aqui/internal/platform/db/dbtest"
	"github.com/thelunardi/inscreve-aqui/internal/platform/migrate"
	"github.com/thelunardi/inscreve-aqui/migrations"
)

// TestUpDownUp garante que todas as migrations sobem, descem e sobem de novo
// em um banco vazio (ADR-0010).
func TestUpDownUp(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dbURL := dbtest.NewEmptyURL(t)

	m, err := migrate.Open(dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	want := len(files)

	up, err := m.Up(ctx)
	if err != nil {
		t.Fatalf("up: %v", err)
	}
	if len(up) != want {
		t.Fatalf("up applied %d migrations, want %d", len(up), want)
	}

	down, err := m.DownAll(ctx)
	if err != nil {
		t.Fatalf("down: %v", err)
	}
	if len(down) != want {
		t.Fatalf("down reverted %d migrations, want %d", len(down), want)
	}
	if n := userTables(t, dbURL); n != 0 {
		t.Fatalf("%d tables left after down, want 0", n)
	}

	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

// userTables conta as tabelas do schema public, exceto a de controle do goose.
func userTables(t *testing.T, dbURL string) int {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	var n int
	err = conn.QueryRow(ctx, `
		SELECT count(*) FROM pg_tables
		WHERE schemaname = 'public' AND tablename <> 'goose_db_version'`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}
