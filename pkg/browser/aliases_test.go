package browser

import "testing"

func TestSeededUserAliases(t *testing.T) {
	const expected = "user-seeded-a813d5642c0c2fea0799d8b05fdb19df"
	users := []map[string]any{{"id": "user-0000000", "email": " Maya@example.com "}}
	got := seededUserAliases(users)
	if len(got) != 1 {
		t.Fatalf("expected one alias: %v", got)
	}
	var key string
	for k, id := range got {
		key = k
		if k != expected || id != "user-0000000" {
			t.Fatalf("unexpected alias: %v", got)
		}
	}
	normalized := seededUserAliases([]map[string]any{{"id": "user-0000000", "email": "maya@example.com"}})
	if normalized[key] != "user-0000000" {
		t.Fatal("email normalization differs")
	}
	users = append(users, map[string]any{"id": "another-user", "email": "maya@example.com"})
	if len(seededUserAliases(users)) != 0 {
		t.Fatal("ambiguous email was resolved")
	}
	users = append(users, map[string]any{"id": key, "email": "different@example.com"})
	if _, ok := seededUserAliases(users)[key]; ok {
		t.Fatal("alias shadows a stored ID")
	}
}
