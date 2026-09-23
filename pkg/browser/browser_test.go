package browser

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conductorone/baton-demo/pkg/client"
	"github.com/conductorone/baton-demo/pkg/config"
)

func TestViewerObservesProvisioning(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "demo.db")
	writer, err := client.NewClient(ctx, &config.Demo{DbFileName: path, Users: 2, Groups: 1, Roles: 1, Projects: 1, ScopedRoles: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writer.Close() })
	u, err := writer.CreateUser(ctx, "Browser test", "browser@example.com", "not-for-display")
	if err != nil {
		t.Fatal(err)
	}
	groups, err := writer.ListGroups(ctx)
	if err != nil || len(groups) == 0 {
		t.Fatalf("groups: %v, %v", groups, err)
	}
	db, abs, err := openDatabase(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	h := handler(db, abs)
	read := func() snapshot {
		t.Helper()
		r := httptest.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/api/snapshot", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("snapshot: %d %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "not-for-display") || strings.Contains(w.Body.String(), "password") {
			t.Fatal("password data exposed")
		}
		var s snapshot
		if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	before := read()
	if strings.Contains(before.Data["groups"][0]["members"].(string), u.Id) {
		t.Fatal("user already a member")
	}
	if err := writer.GrantGroupMember(ctx, groups[0].Id, u.Id); err != nil {
		t.Fatal(err)
	}
	after := read()
	if !strings.Contains(after.Data["groups"][0]["members"].(string), u.Id) {
		t.Fatal("grant not visible")
	}
	if err := writer.RevokeGroupMember(ctx, groups[0].Id, u.Id); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(read().Data["groups"][0]["members"].(string), u.Id) {
		t.Fatal("revoke not visible")
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM users"); err == nil {
		t.Fatal("viewer connection permitted mutation")
	}
	for _, tc := range []struct {
		method, host, origin string
		code                 int
	}{
		{http.MethodPost, "localhost", "", http.StatusMethodNotAllowed},
		{http.MethodGet, "attacker.example", "", http.StatusForbidden},
		{http.MethodGet, "localhost", "https://attacker.example", http.StatusForbidden},
	} {
		r := httptest.NewRequestWithContext(ctx, tc.method, "http://"+tc.host+"/api/snapshot", nil)
		r.Header.Set("Origin", tc.origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.code {
			t.Fatalf("%+v: got %d", tc, w.Code)
		}
	}
}

func TestMissingDatabaseIsNotCreated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	if db, _, err := openDatabase(context.Background(), path); err == nil {
		_ = db.Close()
		t.Fatal("opened missing database")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file created: %v", err)
	}
}

func TestOlderSchemaRemainsUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	w, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.ExecContext(context.Background(), "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT); INSERT INTO users VALUES ('u1', 'Existing user')"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	specialPath := filepath.Join(filepath.Dir(path), "demo ?#.db")
	if err := os.Rename(path, specialPath); err != nil {
		t.Fatal(err)
	}
	path = specialPath
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	db, abs, err := openDatabase(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := readSnapshot(context.Background(), db, abs)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Missing) != len(tables)-1 || len(s.Data["users"]) != 1 {
		t.Fatalf("unexpected older schema snapshot: %+v", s)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("viewer modified database")
	}
}
