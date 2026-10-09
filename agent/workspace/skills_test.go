package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScopedInstructionsAndSkills(t *testing.T) {
	dir := t.TempDir()
	w, _ := New(dir)
	for name, content := range map[string]string{
		"AGENTS.md": "root guidance", "sub/AGENTS.md": "nested guidance", "sub/code.txt": "source",
		".agents/skills/review/SKILL.md": "---\nname: review\ndescription: Review changes\n---\nRead the diff before commenting.",
	} {
		file := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := w.readWithInstructions(context.Background(), map[string]any{"path": "sub/code.txt"})
	if err != nil || !strings.Contains(got, "root guidance") || !strings.Contains(got, "nested guidance") || !strings.HasSuffix(got, "source") {
		t.Fatalf("%q %v", got, err)
	}
	skills, err := w.Skills()
	if err != nil || len(skills) != 1 || skills[0].Name != "review" {
		t.Fatalf("%+v %v", skills, err)
	}
	got, err = w.skill(context.Background(), map[string]any{"name": "review"})
	if err != nil || !strings.Contains(got, "Read the diff") {
		t.Fatalf("%q %v", got, err)
	}
	if _, err = w.InstructionsFor("../escape"); err == nil {
		t.Fatal("instructions escaped workspace")
	}
}

func TestSaveSkillAndReuseLibrary(t *testing.T) {
	project, library := t.TempDir(), t.TempDir()
	w, _ := New(project, WithSkills(library))
	if err := w.SaveSkill("review", "Review changes", "Read the diff."); err != nil {
		t.Fatal(err)
	}
	if err := w.SaveSkill("review", "Overwrite", "No"); err == nil {
		t.Fatal("overwrote existing skill")
	}
	if err := w.SaveSkill("../escape", "Escape", "No"); err == nil {
		t.Fatal("escaped project")
	}
	if err := os.MkdirAll(filepath.Join(library, "shared"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(library, "shared", "SKILL.md"), []byte("---\nname: shared\ndescription: Shared guidance\n---\nShared procedure."), 0600); err != nil {
		t.Fatal(err)
	}
	skills, err := w.Skills()
	if err != nil || len(skills) != 2 {
		t.Fatalf("skills=%v err=%v", skills, err)
	}
	content, err := w.skill(context.Background(), map[string]any{"name": "shared"})
	if err != nil || !strings.Contains(content, "Shared procedure") {
		t.Fatalf("content=%q err=%v", content, err)
	}
}
