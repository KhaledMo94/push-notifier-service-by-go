package command

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	gormlogger "gorm.io/gorm/logger"

	"github.com/KhaledMo94/push-notifier-service-by-go/internal/config"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/crypto"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/db"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/models"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/repository"
)

type createCmd struct{}

func init() { Register(&createCmd{}) }

func (*createCmd) Name() string  { return "create" }
func (*createCmd) Usage() string { return "create a backend and print its API token" }

type CreateArgs struct {
	name, desc, locale, credentialPath string
	workers                            int
}

func (*createCmd) Run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	var a CreateArgs
	fs.StringVar(&a.name, "name", "", "backend name (required, unique)")
	fs.StringVar(&a.desc, "description", "", "optional description")
	fs.IntVar(&a.workers, "workers", 0, "worker count (defaults to DEFAULT_WORKER_COUNT)")
	fs.StringVar(&a.locale, "locale", "", "default locale (defaults to DEFAULT_LOCALE)")
	fs.StringVar(&a.credentialPath, "fcm-credentials", "", "path to the Firebase service-account JSON (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return createBackend(ctx, a)
}

func createBackend(ctx context.Context, a CreateArgs) error {
	if a.name == "" || a.credentialPath == "" {
		return errors.New("missing name or credentials")
	}

	cnf, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if a.workers == 0 {
		a.workers = cnf.DefaultWorkerCount
	}

	if a.workers < 0 {
		return fmt.Errorf("-workers must be > 0, got %d", a.workers)
	}

	if a.locale == "" {
		a.locale = cnf.DefaultLocale
	}

	cipher, err := crypto.NewCipher(cnf.Encryption.Algorithm, cnf.Encryption.Key)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cred, err := os.ReadFile(a.credentialPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	encrypted, err := cipher.Encrypt(cred)
	if err != nil {
		return fmt.Errorf("enc failed: %w", err)
	}

	plainToken, hashToken, err := crypto.GenerateToken()
	if err != nil {
		return err
	}

	dbConn, err := db.Open(cnf.DBPath, gormlogger.Warn)
	if err != nil {
		return fmt.Errorf("db conn failed: %w", err)
	}
	defer dbConn.Close()

	backend := &models.Backend{
		Name:           a.name,
		TokenHash:      hashToken,
		WorkerCount:    a.workers,
		DefaultLocale:  a.locale,
		FCMCredentials: encrypted,
	}

	if a.desc != "" {
		backend.Description = &a.desc
	}

	var writer repository.BackendWriter = repository.NewBackendRepository(dbConn)
	if err := writer.Create(ctx, backend); err != nil {
		return fmt.Errorf("create backend: %w", err)
	}

	fmt.Printf("backend %q created (id %d)\n", backend.Name, backend.ID)
	fmt.Printf("API token (shown once): %s\n", plainToken)
	return nil
}
