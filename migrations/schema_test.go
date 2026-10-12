package migrations_test

// Testes das garantias que o schema impõe sozinho, sem código de aplicação
// (ADR-0003, ADR-0004, ADR-0005, ADR-0011). Rodam contra um PostgreSQL real.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thelunardi/inscreve-aqui/internal/platform/db/dbtest"
)

const (
	codeUniqueViolation     = "23505"
	codeForeignKeyViolation = "23503"
	codeCheckViolation      = "23514"
)

// Dados fictícios: nenhum CPF aqui pertence a uma pessoa real.
const (
	fakeCPF      = "00000000001"
	otherFakeCPF = "00000000002"
)

func TestSameCPFConcurrentlyOnlyOneWins(t *testing.T) {
	t.Parallel()
	pool := dbtest.New(t)
	s := seed(t, pool, 100)

	const attempts = 10
	errs := concurrently(attempts, func() error {
		orderID := s.order(t)
		_, err := pool.Exec(context.Background(), insertRegistration, orderID, s.eventID, s.categoryID, fakeCPF)
		return err
	})

	ok := 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case isViolation(err, codeUniqueViolation, "uq_registrations_event_cpf"):
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if ok != 1 {
		t.Fatalf("%d registrations with the same CPF succeeded, want exactly 1", ok)
	}
}

func TestCanceledRegistrationReleasesCPF(t *testing.T) {
	t.Parallel()
	pool := dbtest.New(t)
	s := seed(t, pool, 100)
	ctx := context.Background()

	var id int64
	if err := pool.QueryRow(ctx, insertRegistration+" RETURNING id", s.order(t), s.eventID, s.categoryID, fakeCPF).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE registrations SET status = 'canceled' WHERE id = $1", id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, insertRegistration, s.order(t), s.eventID, s.categoryID, fakeCPF); err != nil {
		t.Fatalf("new registration after cancel: %v", err)
	}
}

// reserveSlots é o comando do ADR-0004.
const reserveSlots = `
	UPDATE categories
	   SET slots_taken = slots_taken + $2
	 WHERE id = $1 AND slots_taken + $2 <= capacity`

func TestLastSlotsRaceNeverOverbooks(t *testing.T) {
	t.Parallel()
	pool := dbtest.New(t)
	const capacity = 5
	s := seed(t, pool, capacity)
	ctx := context.Background()

	const attempts = 20
	var mu sync.Mutex
	reserved := 0
	errs := concurrently(attempts, func() error {
		tag, err := pool.Exec(ctx, reserveSlots, s.categoryID, 1)
		if err != nil {
			return err
		}
		mu.Lock()
		reserved += int(tag.RowsAffected())
		mu.Unlock()
		return nil
	})
	for _, err := range errs {
		if err != nil {
			t.Fatalf("reserve: %v", err)
		}
	}

	var taken int
	if err := pool.QueryRow(ctx, "SELECT slots_taken FROM categories WHERE id = $1", s.categoryID).Scan(&taken); err != nil {
		t.Fatal(err)
	}
	if reserved != capacity || taken != capacity {
		t.Fatalf("reserved %d, slots_taken %d, want both %d", reserved, taken, capacity)
	}
}

func TestReservingMoreThanAvailableAffectsNoRows(t *testing.T) {
	t.Parallel()
	pool := dbtest.New(t)
	s := seed(t, pool, 3)

	tag, err := pool.Exec(context.Background(), reserveSlots, s.categoryID, 4)
	if err != nil {
		t.Fatal(err)
	}
	if tag.RowsAffected() != 0 {
		t.Fatalf("rows affected = %d, want 0", tag.RowsAffected())
	}
}

// TestConstraints verifica que cada estado proibido é rejeitado pelo banco,
// com o código de erro e o nome da restrição esperados.
func TestConstraints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		code       string
		constraint string
		violate    func(t *testing.T, pool *pgxpool.Pool, s *seeded) error
	}{
		{
			name: "overbooking by any path", code: codeCheckViolation, constraint: "ck_categories_slots",
			violate: func(t *testing.T, pool *pgxpool.Pool, s *seeded) error {
				_, err := pool.Exec(context.Background(), "UPDATE categories SET slots_taken = capacity + 1 WHERE id = $1", s.categoryID)
				return err
			},
		},
		{
			name: "bib number reused after cancel", code: codeUniqueViolation, constraint: "uq_registrations_event_bib",
			violate: func(t *testing.T, pool *pgxpool.Pool, s *seeded) error {
				ctx := context.Background()
				if _, err := pool.Exec(ctx, insertConfirmed, s.order(t), s.eventID, s.categoryID, fakeCPF, 1, "canceled"); err != nil {
					t.Fatal(err)
				}
				_, err := pool.Exec(ctx, insertConfirmed, s.order(t), s.eventID, s.categoryID, otherFakeCPF, 1, "confirmed")
				return err
			},
		},
		{
			name: "confirmed without bib number", code: codeCheckViolation, constraint: "ck_registrations_confirmed_has_bib",
			violate: func(t *testing.T, pool *pgxpool.Pool, s *seeded) error {
				_, err := pool.Exec(context.Background(), insertConfirmed, s.order(t), s.eventID, s.categoryID, fakeCPF, nil, "confirmed")
				return err
			},
		},
		{
			name: "category from another event", code: codeForeignKeyViolation, constraint: "fk_registrations_category",
			violate: func(t *testing.T, pool *pgxpool.Pool, s *seeded) error {
				other := seed(t, pool, 10)
				_, err := pool.Exec(context.Background(), insertRegistration, s.order(t), s.eventID, other.categoryID, fakeCPF)
				return err
			},
		},
		{
			name: "invalid cpf format", code: codeCheckViolation, constraint: "ck_registrations_cpf",
			violate: func(t *testing.T, pool *pgxpool.Pool, s *seeded) error {
				_, err := pool.Exec(context.Background(), insertRegistration, s.order(t), s.eventID, s.categoryID, "000.000.000")
				return err
			},
		},
		{
			name: "duplicate idempotency key", code: codeUniqueViolation, constraint: "uq_orders_idempotency_key",
			violate: func(t *testing.T, pool *pgxpool.Pool, s *seeded) error {
				ctx := context.Background()
				key := "6f1c1f9e-0000-4000-8000-000000000001"
				if _, err := pool.Exec(ctx, insertOrder, s.eventID, "CODE-A", key); err != nil {
					t.Fatal(err)
				}
				_, err := pool.Exec(ctx, insertOrder, s.eventID, "CODE-B", key)
				return err
			},
		},
		{
			name: "paid order without paid_at", code: codeCheckViolation, constraint: "ck_orders_paid_at",
			violate: func(t *testing.T, pool *pgxpool.Pool, s *seeded) error {
				_, err := pool.Exec(context.Background(), "UPDATE orders SET status = 'paid' WHERE id = $1", s.order(t))
				return err
			},
		},
		{
			name: "duplicate gateway payment", code: codeUniqueViolation, constraint: "uq_payments_gateway_external_id",
			violate: func(t *testing.T, pool *pgxpool.Pool, s *seeded) error {
				ctx := context.Background()
				q := `INSERT INTO payments (order_id, gateway, external_id, amount_cents, expires_at)
				      VALUES ($1, 'fake', 'pay-1', 5000, now() + interval '20 minutes')`
				if _, err := pool.Exec(ctx, q, s.order(t)); err != nil {
					t.Fatal(err)
				}
				_, err := pool.Exec(ctx, q, s.order(t))
				return err
			},
		},
		{
			name: "duplicate webhook", code: codeUniqueViolation, constraint: "uq_received_webhooks_gateway_event",
			violate: func(t *testing.T, pool *pgxpool.Pool, s *seeded) error {
				ctx := context.Background()
				q := `INSERT INTO received_webhooks (gateway, external_event_id, payload) VALUES ('fake', 'evt-1', '{}')`
				if _, err := pool.Exec(ctx, q); err != nil {
					t.Fatal(err)
				}
				_, err := pool.Exec(ctx, q)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pool := dbtest.New(t)
			s := seed(t, pool, 100)
			err := tt.violate(t, pool, s)
			if !isViolation(err, tt.code, tt.constraint) {
				t.Fatalf("err = %v, want %s on %s", err, tt.code, tt.constraint)
			}
		})
	}
}

// --- helpers ---

const (
	insertOrder = `
		INSERT INTO orders (event_id, code, buyer_name, buyer_email, total_cents, idempotency_key, expires_at)
		VALUES ($1, $2, 'Comprador Teste', 'comprador@example.com', 5000, $3, now() + interval '30 minutes')
		RETURNING id`
	insertRegistration = `
		INSERT INTO registrations (order_id, event_id, category_id, name, cpf, birth_date)
		VALUES ($1, $2, $3, 'Participante Teste', $4, '1990-01-01')`
	insertConfirmed = `
		INSERT INTO registrations (order_id, event_id, category_id, name, cpf, birth_date, bib_number, status)
		VALUES ($1, $2, $3, 'Participante Teste', $4, '1990-01-01', $5, $6)`
)

type seeded struct {
	pool       *pgxpool.Pool
	eventID    int64
	categoryID int64

	mu     sync.Mutex
	orders int
}

// seed cria um evento com uma categoria de capacidade capacity.
func seed(t *testing.T, pool *pgxpool.Pool, capacity int) *seeded {
	t.Helper()
	ctx := context.Background()
	s := &seeded{pool: pool}
	err := pool.QueryRow(ctx,
		`INSERT INTO events (name, starts_at) VALUES ('Pedal Teste', now() + interval '30 days') RETURNING id`,
	).Scan(&s.eventID)
	if err != nil {
		t.Fatal(err)
	}
	err = pool.QueryRow(ctx,
		`INSERT INTO categories (event_id, name, price_cents, capacity) VALUES ($1, 'Geral', 5000, $2) RETURNING id`,
		s.eventID, capacity,
	).Scan(&s.categoryID)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// order cria um pedido novo no evento. Seguro para uso concorrente.
func (s *seeded) order(t *testing.T) int64 {
	s.mu.Lock()
	s.orders++
	n := s.orders
	s.mu.Unlock()

	var id int64
	code := fmt.Sprintf("E%d-O%d", s.eventID, n)
	key := fmt.Sprintf("00000000-0000-4000-8000-%012d", s.eventID*1_000_000+int64(n))
	if err := s.pool.QueryRow(context.Background(), insertOrder, s.eventID, code, key).Scan(&id); err != nil {
		t.Error(err)
	}
	return id
}

// concurrently roda fn n vezes em goroutines liberadas ao mesmo tempo.
func concurrently(n int, fn func() error) []error {
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = fn()
		}()
	}
	close(start)
	wg.Wait()
	return errs
}

func isViolation(err error, code, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code && pgErr.ConstraintName == constraint
}
