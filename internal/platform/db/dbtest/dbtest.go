// Package dbtest cria bancos PostgreSQL isolados para testes.
//
// Cada banco é criado no servidor de TEST_DATABASE_URL e apagado ao fim do
// teste, então testes podem rodar em paralelo sem interferir uns nos outros.
package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thelunardi/inscreve-aqui/internal/platform/migrate"
)

// defaultURL aponta para o PostgreSQL do docker-compose.yml.
const defaultURL = "postgres://inscreve:inscreve@localhost:5432/postgres?sslmode=disable"

// New cria um banco com todas as migrations aplicadas e devolve um pool
// conectado a ele.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	dbURL := NewEmptyURL(t)

	m, err := migrate.Open(dbURL)
	if err != nil {
		t.Fatalf("dbtest: open migrator: %v", err)
	}
	defer m.Close()
	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("dbtest: apply migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("dbtest: open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// NewEmptyURL cria um banco vazio, sem migrations, e devolve a URL de conexão.
func NewEmptyURL(t testing.TB) string {
	t.Helper()
	ctx := context.Background()

	serverURL := os.Getenv("TEST_DATABASE_URL")
	if serverURL == "" {
		serverURL = defaultURL
	}

	admin, err := pgx.Connect(ctx, serverURL)
	if err != nil {
		t.Fatalf("dbtest: connect to test server (is it running? try `make db-up`): %v", err)
	}
	defer admin.Close(ctx)

	name := "test_" + randomSuffix(t)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("dbtest: create database: %v", err)
	}

	// Registrado antes dos Cleanup de quem chama, então roda depois deles
	// (Cleanup é LIFO): o banco só é apagado quando os pools já foram fechados.
	t.Cleanup(func() {
		conn, err := pgx.Connect(ctx, serverURL)
		if err != nil {
			t.Errorf("dbtest: connect to drop database: %v", err)
			return
		}
		defer conn.Close(ctx)
		if _, err := conn.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("dbtest: drop database: %v", err)
		}
	})

	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("dbtest: parse url: %v", err)
	}
	u.Path = "/" + name
	return u.String()
}

func randomSuffix(t testing.TB) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("dbtest: random: %v", err)
	}
	return hex.EncodeToString(b)
}
