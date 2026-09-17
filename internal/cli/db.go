package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/misbakhul29/goks/pkg/orm"
	"github.com/spf13/cobra"
)

// DBCmd returns the `goks db` command group.
func DBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database migration and management commands",
	}

	cmd.AddCommand(dbMakeMigrationCmd())
	cmd.AddCommand(dbMigrateCmd())
	cmd.AddCommand(dbRollbackCmd())
	cmd.AddCommand(dbStatusCmd())
	cmd.AddCommand(dbSeedCmd())
	return cmd
}

func connectDB(appDir string) (*orm.Database, error) {
	if appDir == "" {
		appDir = "."
	}
	_ = godotenv.Load(filepath.Join(appDir, ".env"))

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		if strings.HasPrefix(dbURL, "postgres://") || strings.HasPrefix(dbURL, "postgresql://") {
			return orm.Connect("postgres", dbURL)
		}
		if strings.HasPrefix(dbURL, "mysql://") {
			return orm.Connect("mysql", strings.TrimPrefix(dbURL, "mysql://"))
		}
		if strings.HasPrefix(dbURL, "sqlite://") {
			return orm.OpenSQLite(filepath.Join(appDir, strings.TrimPrefix(dbURL, "sqlite://")))
		}
	}

	// Check driver + db name
	driver := os.Getenv("DB_DRIVER")
	dbName := os.Getenv("DB_NAME")
	if driver == "postgres" && dbName != "" {
		return orm.Connect("postgres", fmt.Sprintf("dbname=%s sslmode=disable", dbName))
	}

	// Default to SQLite in app directory
	sqlitePath := filepath.Join(appDir, "app.db")
	return orm.OpenSQLite(sqlitePath)
}

var validMigrationName = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func resolveMigrationsDir(appDir string, forCreation bool) string {
	if appDir == "" {
		appDir = "."
	}
	// 1. If database/migrations exists, prioritize it
	dbMig := filepath.Join(appDir, "database", "migrations")
	if info, err := os.Stat(dbMig); err == nil && info.IsDir() {
		return dbMig
	}
	// 2. If db/migrations exists, prioritize it
	shortDbMig := filepath.Join(appDir, "db", "migrations")
	if info, err := os.Stat(shortDbMig); err == nil && info.IsDir() {
		return shortDbMig
	}
	if forCreation {
		// If database/ directory exists, create migrations inside database/migrations
		if info, err := os.Stat(filepath.Join(appDir, "database")); err == nil && info.IsDir() {
			return dbMig
		}
		// If db/ directory exists, create migrations inside db/migrations
		if info, err := os.Stat(filepath.Join(appDir, "db")); err == nil && info.IsDir() {
			return shortDbMig
		}
	}
	// 3. Fallback to migrations/
	return filepath.Join(appDir, "migrations")
}

func resolveSeedsDir(appDir string) string {
	if appDir == "" {
		appDir = "."
	}
	dbSeeds := filepath.Join(appDir, "database", "seeds")
	if info, err := os.Stat(dbSeeds); err == nil && info.IsDir() {
		return dbSeeds
	}
	shortDbSeeds := filepath.Join(appDir, "db", "seeds")
	if info, err := os.Stat(shortDbSeeds); err == nil && info.IsDir() {
		return shortDbSeeds
	}
	return filepath.Join(appDir, "seeds")
}

func dbMakeMigrationCmd() *cobra.Command {
	var appDir string

	cmd := &cobra.Command{
		Use:     "make:migration <name>",
		Aliases: []string{"make", "create"},
		Short:   "Create a new timestamped SQL migration file",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if !validMigrationName.MatchString(name) {
				return fmt.Errorf("invalid migration name: only alphanumeric characters and underscores are allowed")
			}

			if appDir == "" {
				appDir = "."
			}

			migrationsDir := resolveMigrationsDir(appDir, true)
			if err := os.MkdirAll(migrationsDir, 0755); err != nil {
				return fmt.Errorf("failed to create migrations directory: %w", err)
			}

			timestamp := time.Now().Format("20060102150405")
			fileName := fmt.Sprintf("%s_%s.sql", timestamp, name)
			targetFile := filepath.Join(migrationsDir, fileName)

			template := `-- +goks Up
-- Write your migration SQL here:


-- +goks Down
-- Write SQL to revert your migration here:

`
			if err := os.WriteFile(targetFile, []byte(template), 0644); err != nil {
				return fmt.Errorf("failed to create migration file: %w", err)
			}

			relDisplay, err := filepath.Rel(appDir, targetFile)
			if err != nil {
				relDisplay = targetFile
			}
			fmt.Println(color.GreenString("✓ Created migration:"), relDisplay)
			return nil
		},
	}

	cmd.Flags().StringVarP(&appDir, "dir", "d", "", "Application root directory (default: current dir)")
	return cmd
}

func dbMigrateCmd() *cobra.Command {
	var appDir string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run all pending database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			if appDir == "" {
				appDir = "."
			}

			db, err := connectDB(appDir)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer db.Close()

			migrationsDir := resolveMigrationsDir(appDir, false)
			migrations, err := orm.LoadMigrationsFromDir(migrationsDir)
			if err != nil {
				return err
			}

			if len(migrations) == 0 {
				relDir, _ := filepath.Rel(appDir, migrationsDir)
				fmt.Printf("%s\n", color.YellowString("No migration files found in "+relDir+" directory."))
				return nil
			}

			fmt.Println(color.CyanString("\n  🚀 Running GoKS Database Migrations:"))
			fmt.Println()

			start := time.Now()
			if err := orm.Migrate(db, migrations); err != nil {
				return err
			}

			fmt.Printf("\n  %s (took %v)\n\n", color.GreenString("✨ All migrations applied successfully!"), time.Since(start))
			return nil
		},
	}

	cmd.Flags().StringVarP(&appDir, "dir", "d", "", "Application root directory (default: current dir)")
	return cmd
}

func dbRollbackCmd() *cobra.Command {
	var appDir string
	var steps int

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback the last batch of database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			if appDir == "" {
				appDir = "."
			}

			db, err := connectDB(appDir)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer db.Close()

			migrationsDir := resolveMigrationsDir(appDir, false)
			migrations, err := orm.LoadMigrationsFromDir(migrationsDir)
			if err != nil {
				return err
			}

			start := time.Now()
			if err := orm.Rollback(db, migrations, steps); err != nil {
				return err
			}

			fmt.Printf("\n  %s (took %v)\n\n", color.GreenString("✨ Rollback completed successfully!"), time.Since(start))
			return nil
		},
	}

	cmd.Flags().StringVarP(&appDir, "dir", "d", "", "Application root directory (default: current dir)")
	cmd.Flags().IntVarP(&steps, "steps", "s", 1, "Number of batches to rollback (default: 1)")
	return cmd
}

func dbStatusCmd() *cobra.Command {
	var appDir string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of all database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			if appDir == "" {
				appDir = "."
			}

			db, err := connectDB(appDir)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer db.Close()

			migrationsDir := resolveMigrationsDir(appDir, false)
			migrations, err := orm.LoadMigrationsFromDir(migrationsDir)
			if err != nil {
				return err
			}

			statuses, err := orm.GetStatus(db, migrations)
			if err != nil {
				return err
			}

			fmt.Println(color.CyanString("\n  📊 GoKS Migration Status:"))
			fmt.Println()
			fmt.Printf("  %-10s | %-16s | %-32s | %-6s\n", "Status", "Version", "Name", "Batch")
			fmt.Println("  -----------+------------------+----------------------------------+-------")

			for _, s := range statuses {
				statusText := color.YellowString("Pending")
				batchStr := "-"
				if s.Applied {
					statusText = color.GreenString("Applied")
					batchStr = fmt.Sprintf("%d", s.Batch)
				}
				fmt.Printf("  %-10s | %-16s | %-32s | %-6s\n", statusText, s.Version, s.Name, batchStr)
			}
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVarP(&appDir, "dir", "d", "", "Application root directory (default: current dir)")
	return cmd
}

func dbSeedCmd() *cobra.Command {
	var appDir string

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Run database seeders from seeds/ directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if appDir == "" {
				appDir = "."
			}

			db, err := connectDB(appDir)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer db.Close()

			seedsDir := resolveSeedsDir(appDir)
			executed, err := orm.RunSeeders(db, seedsDir)
			if err != nil {
				return err
			}

			if len(executed) == 0 {
				relDir, _ := filepath.Rel(appDir, seedsDir)
				fmt.Printf("%s\n", color.YellowString("No seed files (.sql) found in "+relDir+" directory."))
				return nil
			}

			fmt.Println(color.CyanString("\n  🌱 Executed Seeders:"))
			for _, file := range executed {
				fmt.Printf("  %s %s\n", color.GreenString("✓"), file)
			}
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVarP(&appDir, "dir", "d", "", "Application root directory (default: current dir)")
	return cmd
}
