package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raja-aiml/air/internal/engine"
	db "github.com/raja-aiml/air/internal/foundation/database"
)

// DBCommands holds dependencies for database commands.
type DBCommands struct {
	databaseURL string
}

// NewDBCommands creates database command handlers.
func NewDBCommands(databaseURL string) *DBCommands {
	return &DBCommands{databaseURL: databaseURL}
}

// Register adds all database commands to the registry.
func (c *DBCommands) Register(r *engine.Registry) {
	r.Register(&engine.Command{
		Name:        "db.migrate",
		Description: "Run database migrations",
		Examples: []string{
			"run migrations",
			"migrate database",
			"apply migrations",
			"update database schema",
		},
		Parameters: []engine.Parameter{},
		Execute:    c.migrate,
	})

	r.Register(&engine.Command{
		Name:        "db.ping",
		Description: "Check database connectivity",
		Examples: []string{
			"ping database",
			"check database connection",
			"is database running",
			"test database",
		},
		Parameters: []engine.Parameter{},
		Execute:    c.ping,
	})

	r.Register(&engine.Command{
		Name:        "db.query",
		Description: "Execute a SQL query",
		Examples: []string{
			"run query",
			"execute sql",
			"query database",
		},
		Parameters: []engine.Parameter{
			{Name: "sql", Type: "string", Required: true, Description: "SQL query to execute"},
		},
		Execute: c.query,
	})

	r.Register(&engine.Command{
		Name:        "db.shell",
		Description: "Start interactive SQL shell (pure Go, no psql required)",
		Examples: []string{
			"open database shell",
			"start sql shell",
			"database console",
			"psql shell",
		},
		Parameters: []engine.Parameter{},
		Execute:    c.shell,
	})
}

// withPool creates a database connection pool and passes it to the given function.
// It handles pool creation, error handling, and cleanup automatically.
func (c *DBCommands) withPool(ctx context.Context, fn func(*pgxpool.Pool) (engine.Result, error)) (engine.Result, error) {
	pool, err := db.NewPool(ctx, c.databaseURL)
	if err != nil {
		return engine.ErrorResult(err), err
	}
	defer pool.Close()
	return fn(pool)
}

func (c *DBCommands) migrate(ctx context.Context, params map[string]any) (engine.Result, error) {
	return c.withPool(ctx, func(pool *pgxpool.Pool) (engine.Result, error) {
		if err := db.RunMigrations(ctx, pool); err != nil {
			return engine.ErrorResult(err), err
		}
		return engine.NewResult("Migrations applied successfully"), nil
	})
}

func (c *DBCommands) ping(ctx context.Context, params map[string]any) (engine.Result, error) {
	return c.withPool(ctx, func(pool *pgxpool.Pool) (engine.Result, error) {
		if err := db.Ping(ctx, pool); err != nil {
			return engine.ErrorResult(err), err
		}
		return engine.NewResult("Database connection successful"), nil
	})
}

func (c *DBCommands) query(ctx context.Context, params map[string]any) (engine.Result, error) {
	p := engine.Params(params)
	sql, err := p.StringRequired("sql")
	if err != nil {
		return engine.ErrorResult(err), err
	}

	return c.withPool(ctx, func(pool *pgxpool.Pool) (engine.Result, error) {
		result, err := executeQuery(ctx, pool, sql)
		if err != nil {
			return engine.ErrorResult(err), err
		}
		return engine.NewResultWithData("Query executed", result), nil
	})
}

func (c *DBCommands) shell(ctx context.Context, params map[string]any) (engine.Result, error) {
	return c.withPool(ctx, func(pool *pgxpool.Pool) (engine.Result, error) {
		fmt.Println("Connected to database. Type SQL queries, or 'exit' to quit.")
		fmt.Println("-----------------------------------------------------------")

		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("sql> ")
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.ToLower(line) == "exit" || strings.ToLower(line) == "quit" || strings.ToLower(line) == "\\q" {
				break
			}

			result, err := executeQuery(ctx, pool, line)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			printQueryResult(result)
		}

		return engine.NewResult("Shell session ended"), nil
	})
}

// QueryResult holds the result of a SQL query.
type QueryResult struct {
	Columns      []string        `json:"columns"`
	Rows         [][]interface{} `json:"rows"`
	RowsAffected int64           `json:"rows_affected"`
}

func executeQuery(ctx context.Context, pool *pgxpool.Pool, sql string) (*QueryResult, error) {
	rows, err := pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column names
	fields := rows.FieldDescriptions()
	columns := make([]string, len(fields))
	for i, f := range fields {
		columns[i] = string(f.Name)
	}

	// Collect rows
	var resultRows [][]interface{}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		resultRows = append(resultRows, values)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &QueryResult{
		Columns:      columns,
		Rows:         resultRows,
		RowsAffected: int64(len(resultRows)),
	}, nil
}

func printQueryResult(result *QueryResult) {
	if len(result.Columns) == 0 {
		fmt.Printf("Query OK, %d rows affected\n", result.RowsAffected)
		return
	}

	// Calculate column widths
	widths := make([]int, len(result.Columns))
	for i, col := range result.Columns {
		widths[i] = len(col)
	}
	for _, row := range result.Rows {
		for i, val := range row {
			s := fmt.Sprintf("%v", val)
			if len(s) > widths[i] {
				widths[i] = len(s)
			}
		}
	}

	// Print header
	for i, col := range result.Columns {
		fmt.Printf("%-*s  ", widths[i], col)
	}
	fmt.Println()

	// Print separator
	for i := range result.Columns {
		fmt.Print(strings.Repeat("-", widths[i]) + "  ")
	}
	fmt.Println()

	// Print rows
	for _, row := range result.Rows {
		for i, val := range row {
			fmt.Printf("%-*v  ", widths[i], val)
		}
		fmt.Println()
	}

	fmt.Printf("(%d rows)\n", len(result.Rows))
}
