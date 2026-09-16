package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conductorone/baton-demo/pkg/config"
)

func TestLoadSeededUsers(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "users.csv")
	csv := "employee_id,email,first_name,last_name,display_name,department,employment_status\nE1000,maya.chen@example.com,Maya,Chen,Maya Chen,Executive,Active\nE1001,priya.shah@example.com,Priya,Shah,Priya Shah,Engineering,Inactive\n"
	if err := os.WriteFile(path, []byte(csv), 0o600); err != nil {
		t.Fatal(err)
	}

	users, err := loadSeededUsers(path)
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
		{"duplicate name", "email,display_name\nmaya@example.com,Maya Chen\nother@example.com,Maya Chen\n", "row 3 has duplicate name \"Maya Chen\" (also in row 2)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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
		config:      &config.Demo{},
		seededUsers: []*User{{Name: "Maya Chen", Email: "maya@example.com"}, {Name: "Priya Shah", Email: "priya@example.com"}},
	}
	if got, want := generator.userCount(), 2; got != want {
		t.Fatalf("userCount() = %d, want %d", got, want)
	}
	if _, ok := generator.Next(); !ok { // Everyone group
		t.Fatal("Next() did not return the Everyone group")
	}
	for i, want := range generator.seededUsers {
		resource, ok := generator.Next()
		if !ok || resource.User == nil {
			t.Fatalf("Next() = %#v, %t; want seeded user", resource, ok)
		}
		if resource.User.Id != userId(i) || resource.User.Name != want.Name || resource.User.Email != want.Email {
			t.Fatalf("seeded user = %#v, want id %q, name %q, email %q", resource.User, userId(i), want.Name, want.Email)
		}
	}
}
