package client

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conductorone/baton-demo/pkg/config"
	_ "modernc.org/sqlite"
)

func TestLoadSeededUsers(t *testing.T) {
	t.Parallel()
	users, err := loadSeededUsers(filepath.Join("..", "..", "examples", "users.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(users), 2; got != want {
		t.Fatalf("loaded %d users, want %d", got, want)
	}
	if users[0].Name != "Maya Chen" || users[0].Email != "maya.chen@example.com" || !users[0].Enabled {
		t.Fatalf("unexpected first user: %#v", users[0])
	}
	if users[1].Enabled || users[1].Attrs["department"] != "Engineering" {
		t.Fatalf("unexpected second user: %#v", users[1])
	}
}

func TestLoadSeededUsersValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		csv  string
		want string
	}{
		{"missing email column", "display_name\nMaya Chen\n", "must contain an email column"},
		{"empty email", "email,display_name\n,Maya Chen\n", "row 2 has an empty email"},
		{"missing name", "email,department\nmaya@example.com,Engineering\n", "row 2 has no display_name or first_name/last_name"},
		{"header only", "email,display_name\n", "contains no users"},
		{"wrong column count", "email,display_name\nmaya@example.com\n", "row 2 has 1 columns; expected 2"},
		{"duplicate email", "email,display_name\nmaya@example.com,Maya Chen\nMAYA@example.com,Other Maya\n", "row 3 has duplicate email \"MAYA@example.com\" (also in row 2)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "users.csv")
			if err := os.WriteFile(path, []byte(test.csv), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := loadSeededUsers(path)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("loadSeededUsers() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestLoadSeededUsersDisambiguatesDuplicateNames(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "users.csv")
	csv := "email,display_name\nmaya@example.com,Maya Chen\nother@example.com,Maya Chen\n"
	if err := os.WriteFile(path, []byte(csv), 0o600); err != nil {
		t.Fatal(err)
	}
	users, err := loadSeededUsers(path)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"Maya Chen (maya@example.com)", "Maya Chen (other@example.com)"} {
		if got := users[i].Name; got != want {
			t.Fatalf("user %d name = %q, want %q", i, got, want)
		}
	}
}

func TestSeededUserIDIsStableAcrossRowOrder(t *testing.T) {
	t.Parallel()
	paths := []string{filepath.Join(t.TempDir(), "first.csv"), filepath.Join(t.TempDir(), "second.csv")}
	files := []string{
		"email,display_name\nmaya@example.com,Maya Chen\npriya@example.com,Priya Shah\n",
		"email,display_name\npriya@example.com,Priya Shah\nmaya@example.com,Maya Chen\n",
	}
	ids := make(map[string]string)
	for i, path := range paths {
		if err := os.WriteFile(path, []byte(files[i]), 0o600); err != nil {
			t.Fatal(err)
		}
		users, err := loadSeededUsers(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, user := range users {
			if previous, ok := ids[user.Email]; ok && previous != user.Id {
				t.Fatalf("ID for %s changed from %q to %q", user.Email, previous, user.Id)
			}
			ids[user.Email] = user.Id
		}
	}
}

func TestInitDBWithReorderedSeededUsers(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	firstCSV := filepath.Join(dir, "first.csv")
	secondCSV := filepath.Join(dir, "second.csv")
	if err := os.WriteFile(firstCSV, []byte("email,display_name\nalice@example.com,Alex\nbob@example.com,Alex\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondCSV, []byte("email,display_name\nbob@example.com,Alex\nalice@example.com,Alex\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	database := filepath.Join(dir, "demo.db")
	for i, csv := range []string{firstCSV, secondCSV} {
		client, err := NewClient(context.Background(), &config.Demo{DbFileName: database, InitDb: true, UsersCsv: csv})
		if err != nil {
			t.Fatalf("NewClient() with %s: %v", csv, err)
		}
		if i == 1 {
			for email, wantName := range map[string]string{"alice@example.com": "Alex (alice@example.com)", "bob@example.com": "Alex (bob@example.com)"} {
				var name string
				if err := client.rawDB.QueryRowContext(context.Background(), "SELECT name FROM users WHERE email = ?", email).Scan(&name); err != nil {
					t.Fatal(err)
				}
				if name != wantName {
					t.Fatalf("name for %s = %q, want %q", email, name, wantName)
				}
			}
		}
		if err := client.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadSeededUsersDefaultsStatusToEnabled(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "users.csv")
	if err := os.WriteFile(path, []byte("email,display_name\nmaya@example.com,Maya Chen\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	users, err := loadSeededUsers(path)
	if err != nil {
		t.Fatal(err)
	}
	if !users[0].Enabled {
		t.Fatal("user without employment_status should be enabled")
	}
}

func TestGeneratorReturnsSeededUsers(t *testing.T) {
	t.Parallel()
	generator := generator{
		config: &config.Demo{},
		seededUsers: []*User{
			{Id: seededUserID("maya@example.com"), Name: "Maya Chen", Email: "maya@example.com"},
			{Id: seededUserID("priya@example.com"), Name: "Priya Shah", Email: "priya@example.com"},
		},
	}
	if got, want := generator.userCount(), 2; got != want {
		t.Fatalf("userCount() = %d, want %d", got, want)
	}
	if _, ok := generator.Next(); !ok { // Everyone group
		t.Fatal("Next() did not return the Everyone group")
	}
	for _, want := range generator.seededUsers {
		resource, ok := generator.Next()
		if !ok || resource.User == nil {
			t.Fatalf("Next() = %#v, %t; want seeded user", resource, ok)
		}
		if resource.User.Id != want.Id || resource.User.Name != want.Name || resource.User.Email != want.Email {
			t.Fatalf("seeded user = %#v, want id %q, name %q, email %q", resource.User, want.Id, want.Name, want.Email)
		}
	}
}
