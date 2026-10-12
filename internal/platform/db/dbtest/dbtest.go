// Package dbtest cria bancos PostgreSQL isolados para testes.
//
// Cada chamada a New cria um banco novo no servidor de TEST_DATABASE_URL e o
// apaga ao fim do teste, então testes podem rodar em paralelo sem interferir
// uns nos outros.
package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// defaultURL aponta para o PostgreSQL do docker-compose.yml.
const defaultURL = "postgres://inscreve:inscreve@localhost:5432/postgres?sslmode=disable"

// New cria um banco vazio e devolve um pool conectado a ele.
func New(t testing.TB) *pgxpool.Pool {
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

	cfg, err := pgxpool.ParseConfig(serverURL)
	if err != nil {
		t.Fatalf("dbtest: parse url: %v", err)
	}
	cfg.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("dbtest: open pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
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
	return pool
}

func randomSuffix(t testing.TB) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("dbtest: random: %v", err)
	}
	return hex.EncodeToString(b)
}
