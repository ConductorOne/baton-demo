package client

import (
	"os"
	"path/filepath"
	"testing"
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
