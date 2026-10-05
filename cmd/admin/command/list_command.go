package command

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	gormlogger "gorm.io/gorm/logger"

	"github.com/KhaledMo94/push-notifier-service-by-go/internal/config"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/db"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/models"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/repository"
)

type listCmd struct{}

func init() { Register(&listCmd{}) }

type listArgs struct {
	sort   bool
	search string
}

func (*listCmd) Name() string  { return "list" }
func (*listCmd) Usage() string { return "list all backends" }
func (*listCmd) Run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	var a listArgs
	fs.BoolVar(&a.sort, "sort", false, "sort by newest first (default: by id)")
	fs.StringVar(&a.search, "search", "", "only show backends whose name contains this text (case-insensitive)")

	if err := fs.Parse(args); err != nil {
		return err
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

	return listBackends(ctx, repository.NewBackendRepository(dbConn), a)
}

// listBackends never prints the token hash or the FCM credentials.
func listBackends(ctx context.Context, loader repository.BackendLoader, a listArgs) error {
	backends, err := loader.LoadAll(ctx)
	if err != nil {
		return fmt.Errorf("load backends: %w", err)
	}

	backends = filterByName(backends, a.search)
	if a.sort {
		sort.SliceStable(backends, func(i, j int) bool {
			return backends[i].CreatedAt.After(backends[j].CreatedAt)
		})
	}

	if len(backends) == 0 {
		fmt.Println("no backends found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tWORKERS\tLOCALE\tCREATED\tDESCRIPTION")
	for _, b := range backends {
		desc := ""
		if b.Description != nil {
			desc = *b.Description
		}
		fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s\n",
			b.ID, b.Name, b.WorkerCount, b.DefaultLocale, b.CreatedAt.Format("2006-01-02 15:04"), desc)
	}
	return w.Flush()
}

func filterByName(backends []models.Backend, search string) []models.Backend {
	search = strings.ToLower(strings.TrimSpace(search))
	if search == "" {
		return backends
	}
	filtered := make([]models.Backend, 0, len(backends))
	for _, b := range backends {
		if strings.Contains(strings.ToLower(b.Name), search) {
			filtered = append(filtered, b)
		}
	}
	return filtered
}
