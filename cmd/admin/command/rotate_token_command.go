package command

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/KhaledMo94/push-notifier-service-by-go/internal/config"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/crypto"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/db"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/models"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/repository"
	"gorm.io/gorm/logger"
)

type rotateTokenCmd struct{}

func init() { Register(&rotateTokenCmd{}) }

type rotateTokenArgs struct {
	id   int64
	name string
}

func (*rotateTokenCmd) Name() string { return "rotate-token" }
func (*rotateTokenCmd) Usage() string {
	return "generate a new API token for a backend, replacing the old one"
}

func (*rotateTokenCmd) Run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("rotate-token", flag.ContinueOnError)
	var a rotateTokenArgs
	fs.Int64Var(&a.id, "id", 0, "Backend id in the database (req. if name not provided)")
	fs.StringVar(&a.name, "name", "", "backend name req. if id not provided")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cnf, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	dbConn, err := db.Open(cnf.DBPath, logger.Warn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer dbConn.Close()

	repo := repository.NewBackendRepository(dbConn)
	return rotateToken(ctx, repo, repo, a)
}

func rotateToken(ctx context.Context, finder repository.BackendFinder, writer repository.BackendWriter, a rotateTokenArgs) error {
	backend, err := findBackend(ctx, finder, a)
	if err != nil {
		return err
	}

	plain, hash, err := crypto.GenerateToken()
	if err != nil {
		return err
	}
	backend.TokenHash = hash

	if err := writer.Update(ctx, backend); err != nil {
		return fmt.Errorf("update backend: %w", err)
	}

	fmt.Printf("token for backend %q (id %d) rotated; the old token no longer works\n", backend.Name, backend.ID)
	fmt.Printf("new API token (shown once): %s\n", plain)
	return nil
}

func findBackend(ctx context.Context, finder repository.BackendFinder, a rotateTokenArgs) (*models.Backend, error) {
	var (
		b   *models.Backend
		err error
	)

	if a.id != 0 {
		b, err = finder.GetByID(ctx, a.id)
	} else if a.name != "" {
		b, err = finder.GetByName(ctx, a.name)
	} else {
		return nil, errors.New("id or name must be provided")
	}

	if errors.Is(err, repository.ErrNotFound) {
		return nil, errors.New("backend not found")
	}

	if err != nil {
		return nil, fmt.Errorf("find backend: %w", err)
	}
	return b, nil
}
