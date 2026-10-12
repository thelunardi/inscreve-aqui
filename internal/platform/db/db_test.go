package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thelunardi/inscreve-aqui/internal/platform/db"
	"github.com/thelunardi/inscreve-aqui/internal/platform/db/dbtest"
)

func setup(t *testing.T) (*pgxpool.Pool, *db.Transactor) {
	t.Helper()
	pool := dbtest.New(t)
	if _, err := pool.Exec(context.Background(), "CREATE TABLE items (name text NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	return pool, db.NewTransactor(pool)
}

func insert(ctx context.Context, tx db.Tx, name string) error {
	_, err := db.PgxTx(tx).Exec(ctx, "INSERT INTO items (name) VALUES ($1)", name)
	return err
}

func count(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM items").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestWithinTxCommitsOnSuccess(t *testing.T) {
	t.Parallel()
	pool, tr := setup(t)

	err := tr.WithinTx(context.Background(), func(ctx context.Context, tx db.Tx) error {
		if err := insert(ctx, tx, "a"); err != nil {
			return err
		}
		return insert(ctx, tx, "b")
	})
	if err != nil {
		t.Fatalf("WithinTx: %v", err)
	}
	if got := count(t, pool); got != 2 {
		t.Fatalf("rows = %d, want 2", got)
	}
}

func TestWithinTxRollsBackOnError(t *testing.T) {
	t.Parallel()
	pool, tr := setup(t)
	errBoom := errors.New("boom")

	err := tr.WithinTx(context.Background(), func(ctx context.Context, tx db.Tx) error {
		if err := insert(ctx, tx, "a"); err != nil {
			return err
		}
		return errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want %v", err, errBoom)
	}
	if got := count(t, pool); got != 0 {
		t.Fatalf("rows = %d, want 0 after rollback", got)
	}
}

func TestWithinTxRollsBackOnPanic(t *testing.T) {
	t.Parallel()
	pool, tr := setup(t)

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("panic was not propagated")
			}
		}()
		_ = tr.WithinTx(context.Background(), func(ctx context.Context, tx db.Tx) error {
			if err := insert(ctx, tx, "a"); err != nil {
				return err
			}
			panic("boom")
		})
	}()

	if got := count(t, pool); got != 0 {
		t.Fatalf("rows = %d, want 0 after rollback", got)
	}
}
