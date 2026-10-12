// Package db concentra o acesso ao PostgreSQL compartilhado pelos módulos:
// o pool de conexões e as transações (Unit of Work, ADR-0008).
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect abre um pool de conexões e confirma que o banco responde.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// Tx é uma transação aberta. É opaca: casos de uso apenas a recebem e repassam
// aos repositórios. Só os adaptadores postgres extraem o pgx.Tx, com PgxTx.
//
// O método não exportado impede que outros pacotes criem uma Tx.
type Tx interface {
	pgxTx() pgx.Tx
}

type tx struct{ pgx.Tx }

func (t tx) pgxTx() pgx.Tx { return t.Tx }

// PgxTx devolve a transação do pgx por trás de tx.
// Uso restrito aos adaptadores postgres (verificado pelo teste de arquitetura).
func PgxTx(t Tx) pgx.Tx { return t.pgxTx() }

// Transactor abre transações sobre um pool.
type Transactor struct {
	pool *pgxpool.Pool
}

// NewTransactor cria um Transactor sobre pool.
func NewTransactor(pool *pgxpool.Pool) *Transactor {
	return &Transactor{pool: pool}
}

// WithinTx executa fn em uma transação. Faz commit se fn devolver nil e
// rollback se fn devolver erro ou entrar em panic; o panic é propagado.
//
// Transações não são aninhadas: dentro de fn, repasse a tx recebida em vez de
// chamar WithinTx de novo.
func (t *Transactor) WithinTx(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error {
	return pgx.BeginFunc(ctx, t.pool, func(ptx pgx.Tx) error {
		return fn(ctx, tx{ptx})
	})
}
