package command

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	gormlogger "gorm.io/gorm/logger"

	"github.com/KhaledMo94/push-notifier-service-by-go/internal/config"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/db"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/models"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/repository"
)

type deleteArgs struct {
	id   int64
	name string
	yes  bool
}

type deleteCmd struct{}

func init() { Register(&deleteCmd{}) }

func (*deleteCmd) Name() string  { return "delete" }
func (*deleteCmd) Usage() string { return "delete a backend and its stale tokens" }

func (*deleteCmd) Run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	var a deleteArgs
	fs.StringVar(&a.name, "name", "", "backend name required if no id given")
	fs.Int64Var(&a.id, "id", 0, "id in database required if name not provided")
	fs.BoolVar(&a.yes, "yes", false, "skip the confirmation prompt")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if a.id == 0 && a.name == "" {
		return errors.New("provide exactly one of -id or -name")
	}

	cnf, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	dbConn, err := db.Open(cnf.DBPath, gormlogger.Warn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer dbConn.Close()

	repo := repository.NewBackendRepository(dbConn)
	return deleteBackend(ctx, repo, repo, a)
}

func deleteBackend(ctx context.Context, finder repository.BackendFinder, writer repository.BackendWriter, a deleteArgs) error {
	var (
		backend *models.Backend
		err     error
	)
	if a.id != 0 {
		backend, err = finder.GetByID(ctx, a.id)
	} else {
		backend, err = finder.GetByName(ctx, a.name)
	}
	if errors.Is(err, repository.ErrNotFound) {
		return errors.New("backend not found")
	}
	if err != nil {
		return fmt.Errorf("find backend: %w", err)
	}
	if !a.yes && !confirm(fmt.Sprintf("delete backend %q (id %d) and all its stale tokens?", backend.Name, backend.ID)) {
		fmt.Println("aborted")
		return nil
	}
	if err := writer.Delete(ctx, backend.ID); err != nil {
		return fmt.Errorf("delete backend: %w", err)
	}
	fmt.Printf("backend %q (id %d) deleted\n", backend.Name, backend.ID)
	return nil
}

func confirm(question string) bool {
	fmt.Printf("%s [y/N]: ", question)
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}
