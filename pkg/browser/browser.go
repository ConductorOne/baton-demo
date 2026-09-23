package browser

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

//go:embed web/*
var assets embed.FS

type table struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Columns string `json:"-"`
}

var tables = []table{
	{"users", "Users", "id,name,email,enabled,account_type,attrs,created_at,updated_at"},
	{"groups", "Groups", "id,name,admins,members,created_at,updated_at"},
	{"roles", "Roles", "id,name,direct_assignments,group_assignments,created_at,updated_at"},
	{"projects", "Projects", "id,name,owner,group_assignments,created_at,updated_at"},
	{"scoped_roles", "Scoped roles", "id,project_id,role_id,user_assignments,created_at,updated_at"},
	{"nhis", "Non-human identities", "id,name,kind,nhi_type,nhi_detail,created_at,updated_at"},
	{"agents", "Agents", "id,name,status,identity_id,profile,created_at,updated_at"},
	{"secrets", "Credential metadata", "id,name,credential_type,credential_detail,identity_id,created_at,expires_at,last_used_at,updated_at"},
}

type snapshot struct {
	UserAliases map[string]string           `json:"user_aliases"`
	Path        string                      `json:"path"`
	ReadAt      time.Time                   `json:"read_at"`
	Tables      []table                     `json:"tables"`
	Data        map[string][]map[string]any `json:"data"`
	Missing     []string                    `json:"missing"`
}

func openDatabase(ctx context.Context, path string) (*sql.DB, string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, "", fmt.Errorf("open existing demo database: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, "", fmt.Errorf("database must be a regular file: %s", abs)
	}
	u := url.URL{Scheme: "file", Path: abs}
	q := u.Query()
	q.Set("mode", "ro")
	q.Add("_pragma", "query_only(1)")
	q.Add("_pragma", "busy_timeout(1500)")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, "", err
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, "", err
	}
	return db, abs, nil
}

func readSnapshot(ctx context.Context, db *sql.DB, path string) (*snapshot, error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	s := &snapshot{Path: path, ReadAt: time.Now().UTC(), Tables: tables, Data: make(map[string][]map[string]any), Missing: []string{}}
	for _, t := range tables {
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", t.Name).Scan(&exists); err != nil {
			return nil, err
		}
		if exists == 0 {
			s.Missing = append(s.Missing, t.Name)
			s.Data[t.Name] = []map[string]any{}
			continue
		}
		data, err := readTable(ctx, tx, t)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", t.Name, err)
		}
		s.Data[t.Name] = data
	}
	if len(s.Missing) == len(tables) {
		return nil, fmt.Errorf("no demo tables found in %s", path)
	}
	s.UserAliases = seededUserAliases(s.Data["users"])
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s, nil
}

func seededUserAliases(users []map[string]any) map[string]string {
	aliases := make(map[string]string)
	stored := make(map[string]bool)
	ambiguous := make(map[string]bool)
	for _, user := range users {
		id, _ := user["id"].(string)
		stored[id] = true
		email, _ := user["email"].(string)
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" || id == "" {
			continue
		}
		hash := sha256.Sum256([]byte(email))
		alias := fmt.Sprintf("user-seeded-%x", hash[:16])
		if previous, exists := aliases[alias]; exists && previous != id {
			ambiguous[alias] = true
		}
		aliases[alias] = id
	}
	for alias := range aliases {
		if stored[alias] || ambiguous[alias] {
			delete(aliases, alias)
		}
	}
	return aliases
}

func readTable(ctx context.Context, tx *sql.Tx, t table) ([]map[string]any, error) {
	// Only these known columns are exposed, including when opening an older demo schema.
	rows, err := tx.QueryContext(ctx, "SELECT name FROM pragma_table_info(?)", t.Name)
	if err != nil {
		return nil, err
	}
	available := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			_ = rows.Close()
			return nil, err
		}
		available[name] = true
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	columns := []string{}
	for _, name := range strings.Split(t.Columns, ",") {
		if available[name] {
			columns = append(columns, name)
		}
	}
	if !available["id"] {
		return nil, fmt.Errorf("missing id column")
	}
	rows, err = tx.QueryContext(ctx, "SELECT "+strings.Join(columns, ",")+" FROM "+t.Name+" ORDER BY id LIMIT 10001") //nolint:gosec // Identifiers come exclusively from the static table allowlist.
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		record := make(map[string]any)
		for i, key := range columns {
			value := values[i]
			if bytes, ok := value.([]byte); ok {
				value = string(bytes)
			}
			record[key] = value
		}
		result = append(result, record)
		if len(result) > 10000 {
			return nil, fmt.Errorf("demo viewer supports at most 10000 rows per table")
		}
	}
	return result, rows.Err()
}

func handler(db *sql.DB, path string) http.Handler {
	web, _ := fs.Sub(assets, "web")
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(web)))
	mux.HandleFunc("/api/snapshot", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		s, err := readSnapshot(ctx, db, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; frame-ancestors 'none'; base-uri 'none'")
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if host != "127.0.0.1" && host != "localhost" && host != "::1" {
			http.Error(w, "local access only", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host {
			http.Error(w, "cross-origin access denied", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "read-only viewer", http.StatusMethodNotAllowed)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func Command() *cobra.Command {
	var path string
	var port int
	cmd := &cobra.Command{
		Use: "browse", Short: "View the live demo database in a local web UI", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			db, abs, err := openDatabase(cmd.Context(), path)
			if err != nil {
				return err
			}
			defer db.Close()
			if _, err := readSnapshot(cmd.Context(), db, abs); err != nil {
				return err
			}
			var listenConfig net.ListenConfig
			listener, err := listenConfig.Listen(cmd.Context(), "tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				return err
			}
			defer listener.Close()
			cmd.Printf("Demo browser: http://%s\nDatabase: %s (read-only)\n", listener.Addr(), abs)
			server := &http.Server{Handler: handler(db, abs), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
			done := make(chan struct{})
			defer close(done)
			go func() {
				select {
				case <-cmd.Context().Done():
					_ = server.Close()
				case <-done:
				}
			}()
			if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
				return err
			}
			return nil
		},
	}
	defaultPath := os.Getenv("BATON_DB_FILE_NAME")
	if defaultPath == "" {
		defaultPath = "baton-demo.db"
	}
	cmd.Flags().StringVar(&path, "db-file-name", defaultPath, "Existing demo database to view ($BATON_DB_FILE_NAME)")
	cmd.Flags().IntVar(&port, "port", 8080, "Local HTTP port (0 selects an available port)")
	return cmd
}
