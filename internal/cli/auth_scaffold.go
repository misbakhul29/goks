package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// MakeAuthCmd returns the `goks make:auth` command.
func MakeAuthCmd() *cobra.Command {
	var appDir string

	cmd := &cobra.Command{
		Use:     "make:auth",
		Aliases: []string{"auth"},
		Short:   "Scaffold authentication boilerplate (login, register, actions, models, migration)",
		Long: `Scaffold a complete authentication system into your GoKS project.
Generates:
  - database/migrations/<timestamp>_create_users_table.sql
  - database/models/user.go (or models/user.go)
  - app/login/page.gox
  - app/register/page.gox
  - app/login/actions.go
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthScaffold(appDir)
		},
	}

	cmd.Flags().StringVarP(&appDir, "dir", "d", "", "Application root directory (default: current dir)")
	return cmd
}

func runAuthScaffold(appDir string) error {
	if appDir == "" {
		appDir = "."
	}

	fmt.Println(color.CyanString("\n  🚀 Scaffolding GoKS Authentication System...\n"))

	// 1. Determine Model Directory
	modelDir := filepath.Join(appDir, "models")
	if info, err := os.Stat(filepath.Join(appDir, "database", "models")); err == nil && info.IsDir() {
		modelDir = filepath.Join(appDir, "database", "models")
	} else if info, err := os.Stat(filepath.Join(appDir, "database")); err == nil && info.IsDir() {
		modelDir = filepath.Join(appDir, "database", "models")
	} else if info, err := os.Stat(filepath.Join(appDir, "db", "models")); err == nil && info.IsDir() {
		modelDir = filepath.Join(appDir, "db", "models")
	}

	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model dir: %w", err)
	}

	userModelPath := filepath.Join(modelDir, "user.go")
	if err := writeFileIfNotExist(userModelPath, tmplAuthUserModel); err != nil {
		return err
	}
	printCreated(appDir, userModelPath)

	// 2. Migration
	migrationsDir := resolveMigrationsDir(appDir, true)
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations dir: %w", err)
	}

	timestamp := time.Now().Format("20060102150405")
	migrationFile := filepath.Join(migrationsDir, fmt.Sprintf("%s_create_users_table.sql", timestamp))
	if err := writeFileIfNotExist(migrationFile, tmplAuthMigration); err != nil {
		return err
	}
	printCreated(appDir, migrationFile)

	// 3. Login Page (app/login/page.gox)
	loginDir := filepath.Join(appDir, "app", "login")
	if err := os.MkdirAll(loginDir, 0755); err != nil {
		return fmt.Errorf("failed to create app/login dir: %w", err)
	}

	loginPagePath := filepath.Join(loginDir, "page.gox")
	if err := writeFileIfNotExist(loginPagePath, tmplAuthLoginPage); err != nil {
		return err
	}
	printCreated(appDir, loginPagePath)

	// 4. Register Page (app/register/page.gox)
	registerDir := filepath.Join(appDir, "app", "register")
	if err := os.MkdirAll(registerDir, 0755); err != nil {
		return fmt.Errorf("failed to create app/register dir: %w", err)
	}

	registerPagePath := filepath.Join(registerDir, "page.gox")
	if err := writeFileIfNotExist(registerPagePath, tmplAuthRegisterPage); err != nil {
		return err
	}
	printCreated(appDir, registerPagePath)

	// 5. Auth Actions (app/login/actions.go)
	actionsPath := filepath.Join(loginDir, "actions.go")
	if err := writeFileIfNotExist(actionsPath, tmplAuthActions); err != nil {
		return err
	}
	printCreated(appDir, actionsPath)

	fmt.Println(color.GreenString("\n  ✨ Authentication scaffolding complete!"))
	fmt.Println(color.WhiteString("  Next step: run ") + color.CyanString("goks db migrate") + color.WhiteString(" to create the users table.\n"))
	return nil
}

func writeFileIfNotExist(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		// already exists, do not overwrite
		return nil
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func printCreated(appDir, targetFile string) {
	relDisplay, err := filepath.Rel(appDir, targetFile)
	if err != nil {
		relDisplay = targetFile
	}
	fmt.Printf("  %s %s\n", color.GreenString("create"), relDisplay)
}

// -----------------------------------------------------------------------
// Templates for Auth Scaffolding
// -----------------------------------------------------------------------

var tmplAuthUserModel = `package models

import (
	"time"

	"github.com/misbakhul29/goks/pkg/orm"
)

// User represents an application user in the database.
type User struct {
	orm.Model
	Name      string    ` + "`" + `json:"name" orm:"name"` + "`" + `
	Email     string    ` + "`" + `json:"email" orm:"email,unique"` + "`" + `
	Password  string    ` + "`" + `json:"-" orm:"password"` + "`" + `
	Role      string    ` + "`" + `json:"role" orm:"role"` + "`" + `
	CreatedAt time.Time ` + "`" + `json:"created_at" orm:"created_at"` + "`" + `
	UpdatedAt time.Time ` + "`" + `json:"updated_at" orm:"updated_at"` + "`" + `
}
`

var tmplAuthMigration = `-- +goks Up
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    role TEXT DEFAULT 'user',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- +goks Down
DROP TABLE IF EXISTS users;
`

var tmplAuthLoginPage = `package login

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

// Page renders the user login screen.
type Page struct {
	component.ComponentBase
}

func (p *Page) Render() *component.Node {
	return (
		<div class="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-900 py-12 px-4 sm:px-6 lg:px-8">
			<div class="max-w-md w-full space-y-8 bg-white dark:bg-slate-800 p-8 rounded-xl shadow-lg border border-slate-200 dark:border-slate-700">
				<div>
					<h2 class="text-center text-3xl font-extrabold text-slate-900 dark:text-white">
						Sign in to your account
					</h2>
					<p class="mt-2 text-center text-sm text-slate-600 dark:text-slate-400">
						Or <a href="/register" class="font-medium text-indigo-600 hover:text-indigo-500">create a new account</a>
					</p>
				</div>
				<form class="mt-8 space-y-6" method="POST" action="/__goks_action?name=login">
					<div class="rounded-md shadow-sm space-y-4">
						<div>
							<label for="email" class="block text-sm font-medium text-slate-700 dark:text-slate-300">Email address</label>
							<input id="email" name="email" type="email" required class="appearance-none rounded-lg relative block w-full px-3 py-2 border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm" placeholder="user@example.com" />
						</div>
						<div>
							<label for="password" class="block text-sm font-medium text-slate-700 dark:text-slate-300">Password</label>
							<input id="password" name="password" type="password" required class="appearance-none rounded-lg relative block w-full px-3 py-2 border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm" placeholder="••••••••" />
						</div>
					</div>

					<div class="flex items-center justify-between">
						<div class="flex items-center">
							<input id="remember_me" name="remember_me" type="checkbox" value="1" class="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-slate-300 rounded" />
							<label for="remember_me" class="ml-2 block text-sm text-slate-900 dark:text-slate-300">Remember me</label>
						</div>
					</div>

					<div>
						<button type="submit" class="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-lg text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition duration-150">
							Sign In
						</button>
					</div>
				</form>
			</div>
		</div>
	)
}
`

var tmplAuthRegisterPage = `package register

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

// Page renders the registration screen.
type Page struct {
	component.ComponentBase
}

func (p *Page) Render() *component.Node {
	return (
		<div class="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-900 py-12 px-4 sm:px-6 lg:px-8">
			<div class="max-w-md w-full space-y-8 bg-white dark:bg-slate-800 p-8 rounded-xl shadow-lg border border-slate-200 dark:border-slate-700">
				<div>
					<h2 class="text-center text-3xl font-extrabold text-slate-900 dark:text-white">
						Create your account
					</h2>
					<p class="mt-2 text-center text-sm text-slate-600 dark:text-slate-400">
						Already have an account? <a href="/login" class="font-medium text-indigo-600 hover:text-indigo-500">Sign in</a>
					</p>
				</div>
				<form class="mt-8 space-y-6" method="POST" action="/__goks_action?name=register">
					<div class="rounded-md shadow-sm space-y-4">
						<div>
							<label for="name" class="block text-sm font-medium text-slate-700 dark:text-slate-300">Full Name</label>
							<input id="name" name="name" type="text" required class="appearance-none rounded-lg relative block w-full px-3 py-2 border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm" placeholder="John Doe" />
						</div>
						<div>
							<label for="email" class="block text-sm font-medium text-slate-700 dark:text-slate-300">Email address</label>
							<input id="email" name="email" type="email" required class="appearance-none rounded-lg relative block w-full px-3 py-2 border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm" placeholder="user@example.com" />
						</div>
						<div>
							<label for="password" class="block text-sm font-medium text-slate-700 dark:text-slate-300">Password</label>
							<input id="password" name="password" type="password" required class="appearance-none rounded-lg relative block w-full px-3 py-2 border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm" placeholder="••••••••" />
						</div>
					</div>

					<div>
						<button type="submit" class="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-lg text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition duration-150">
							Sign Up
						</button>
					</div>
				</form>
			</div>
		</div>
	)
}
`

var tmplAuthActions = `package login

import (
	"fmt"

	"github.com/misbakhul29/goks/pkg/action"
	"github.com/misbakhul29/goks/pkg/auth"
)

func init() {
	// Login server action
	action.Register("login", func(ctx *action.Context) (any, error) {
		email := ctx.Get("email")
		password := ctx.Get("password")
		remember := ctx.Get("remember_me") == "1"

		if email == "" || password == "" {
			ctx.Flash("error", "Email and password are required")
			return nil, fmt.Errorf("email and password are required")
		}

		// Example: verify credentials against database and get user
		// In production, fetch user from DB and check: auth.CheckPassword(user.Password, password)
		user := &auth.User{
			ID:    1,
			Email: email,
			Name:  email,
			Roles: []string{"user"},
		}

		if _, err := ctx.Login(user, auth.WithRememberMe(remember)); err != nil {
			return nil, err
		}

		return ctx.RedirectIntended("/dashboard")
	})

	// Logout server action
	action.Register("logout", func(ctx *action.Context) (any, error) {
		ctx.Logout()
		return ctx.RedirectIntended("/login")
	})

	// Register server action
	action.Register("register", func(ctx *action.Context) (any, error) {
		name := ctx.Get("name")
		email := ctx.Get("email")
		password := ctx.Get("password")

		if name == "" || email == "" || password == "" {
			ctx.Flash("error", "All fields are required")
			return nil, fmt.Errorf("all fields are required")
		}

		// Hash password: hashedPassword, _ := auth.HashPassword(password)
		user := &auth.User{
			ID:    2,
			Email: email,
			Name:  name,
			Roles: []string{"user"},
		}

		if _, err := ctx.Login(user); err != nil {
			return nil, err
		}

		return ctx.RedirectIntended("/dashboard")
	})
}
`
