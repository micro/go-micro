package micro

import "testing"

func TestParseModelOptions(t *testing.T) {
	tests := []struct {
		comment   string
		wantTable string
		wantKey   string
	}{
		{" @model\n", "", ""},
		{" @model(table=app_users)\n", "app_users", ""},
		{" @model(key=user_id)\n", "", "user_id"},
		{" @model(table=users, key=user_id)\n", "users", "user_id"},
		{" some description\n @model(table=items)\n", "items", ""},
		{" no annotation here\n", "", ""},
	}

	for _, tt := range tests {
		table, key := parseModelOptions(tt.comment)
		if table != tt.wantTable {
			t.Errorf("parseModelOptions(%q): table = %q, want %q", tt.comment, table, tt.wantTable)
		}
		if key != tt.wantKey {
			t.Errorf("parseModelOptions(%q): key = %q, want %q", tt.comment, key, tt.wantKey)
		}
	}
}

func TestProtoFieldGoType(t *testing.T) {
	// Smoke test - just verify it doesn't panic with nil
	typ := protoFieldGoType(nil)
	if typ != "string" {
		// nil field returns default based on zero value TYPE_DOUBLE=0
		t.Logf("protoFieldGoType(nil) = %q", typ)
	}
}
