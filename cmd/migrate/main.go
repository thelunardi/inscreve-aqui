// Command migrate aplica as migrations do banco (ADR-0010).
//
// Roda no deploy, antes da API e do worker. Lê DATABASE_URL do ambiente.
//
//	migrate up       aplica as migrations pendentes
//	migrate status   lista as migrations aplicadas e pendentes
//	migrate down     desfaz a última migration (somente APP_ENV=development)
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/thelunardi/inscreve-aqui/internal/platform/migrate"
)

const usage = "uso: migrate up | status | down"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return errors.New(usage)
	}
	cmd := args[0]
	if cmd != "up" && cmd != "status" && cmd != "down" {
		return errors.New(usage)
	}
	// Em produção, correções são feitas para a frente com uma nova migration.
	if cmd == "down" && os.Getenv("APP_ENV") != "development" {
		return errors.New("down só é permitido com APP_ENV=development")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL não definida")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	m, err := migrate.Open(dbURL)
	if err != nil {
		return err
	}
	defer m.Close()

	switch cmd {
	case "up":
		results, err := m.Up(ctx)
		for _, r := range results {
			fmt.Printf("aplicada  %s (%s)\n", filepath.Base(r.Source.Path), r.Duration)
		}
		if err != nil {
			return err
		}
		if len(results) == 0 {
			fmt.Println("nenhuma migration pendente")
		}
	case "down":
		r, err := m.Down(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("desfeita  %s (%s)\n", filepath.Base(r.Source.Path), r.Duration)
	case "status":
		statuses, err := m.Status(ctx)
		if err != nil {
			return err
		}
		for _, s := range statuses {
			state := "pendente"
			if !s.AppliedAt.IsZero() {
				state = "aplicada em " + s.AppliedAt.Local().Format("2006-01-02 15:04:05")
			}
			fmt.Printf("%-40s %s\n", filepath.Base(s.Source.Path), state)
		}
	}
	return nil
}
