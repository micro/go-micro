package workspace

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill describes project instructions that can be loaded when needed.
type Skill struct {
	root        string
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Path        string `yaml:"-"`
}

// WithSkills adds a read-only skill library. Project skills take precedence.
func WithSkills(dir string) Option {
	return func(w *Workspace) { w.skillDirs = append(w.skillDirs, dir) }
}

// Skills reads bounded SKILL.md files from the project and configured libraries.
// Loading a skill never executes its scripts.
func (w *Workspace) Skills() ([]Skill, error) {
	type source struct{ dir, prefix string }
	dirs := []source{{w.dir, ".agents/skills"}}
	for _, dir := range w.skillDirs {
		dirs = append(dirs, source{dir, "."})
	}
	seen := map[string]bool{}
	var result []Skill
	for _, dir := range dirs {
		skills, err := readSkills(dir.dir, dir.prefix)
		if err != nil {
			return nil, err
		}
		for _, skill := range skills {
			if !seen[skill.Name] {
				seen[skill.Name] = true
				result = append(result, skill)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func readSkills(dir, prefix string) ([]Skill, error) {
	root, err := os.OpenRoot(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), prefix)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var skills []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		file := path.Join(prefix, entry.Name(), "SKILL.md")
		content, err := readText(root, file)
		if err != nil {
			return nil, fmt.Errorf("skill %s: %w", entry.Name(), err)
		}
		content = strings.ReplaceAll(content, "\r\n", "\n")
		if !strings.HasPrefix(content, "---\n") {
			return nil, fmt.Errorf("skill %s has no YAML front matter", entry.Name())
		}
		header, _, found := strings.Cut(content[4:], "\n---\n")
		if !found {
			return nil, fmt.Errorf("skill %s has unclosed front matter", entry.Name())
		}
		var skill Skill
		if err := yaml.Unmarshal([]byte(header), &skill); err != nil {
			return nil, fmt.Errorf("skill %s: %w", entry.Name(), err)
		}
		if skill.Name != entry.Name() || strings.TrimSpace(skill.Description) == "" {
			return nil, fmt.Errorf("skill %s must declare a matching name and description", entry.Name())
		}
		skill.Path = file
		skill.root = dir
		skills = append(skills, skill)
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills, nil
}

func (w *Workspace) skill(ctx context.Context, input map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	name, err := argument(input, "name")
	if err != nil {
		return "", err
	}
	skills, err := w.Skills()
	if err != nil {
		return "", err
	}
	var listing strings.Builder
	for _, skill := range skills {
		if name == skill.Name {
			root, err := os.OpenRoot(skill.root)
			if err != nil {
				return "", err
			}
			defer root.Close()
			return readText(root, skill.Path)
		}
		fmt.Fprintf(&listing, "%s: %s\n", skill.Name, skill.Description)
	}
	if name != "" {
		return "", fmt.Errorf("unknown skill %q", name)
	}
	return listing.String(), nil
}

// InstructionsFor returns root-to-directory AGENTS.md instructions for a file.
// All paths are resolved within the workspace, including symlink checks.
func (w *Workspace) InstructionsFor(file string) (string, error) {
	root, err := os.OpenRoot(w.dir)
	if err != nil {
		return "", err
	}
	defer root.Close()
	file = path.Clean(file)
	if !fs.ValidPath(file) {
		return "", fmt.Errorf("invalid workspace path %q", file)
	}
	dirs := []string{"."}
	dir := path.Dir(file)
	if dir != "." {
		current := ""
		for _, part := range strings.Split(dir, "/") {
			current = path.Join(current, part)
			dirs = append(dirs, current)
		}
	}
	var result strings.Builder
	for _, dir := range dirs {
		name := path.Join(dir, "AGENTS.md")
		content, err := readText(root, name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if result.Len()+len(content) > maxOutput {
			return "", fmt.Errorf("project instructions exceed 64 KiB")
		}
		fmt.Fprintf(&result, "\nInstructions from %s:\n%s\n", name, content)
	}
	return result.String(), nil
}

func (w *Workspace) readWithInstructions(ctx context.Context, input map[string]any) (string, error) {
	content, err := w.read(ctx, input)
	if err != nil {
		return "", err
	}
	file, err := argument(input, "path")
	if err != nil {
		return "", err
	}
	instructions, err := w.InstructionsFor(file)
	if err != nil {
		return "", err
	}
	if instructions == "" {
		return content, nil
	}
	return instructions + "\nFile content:\n" + content, nil
}

// SaveSkill creates a project skill. Existing skills are edited with the normal
// file tools, so saving cannot silently overwrite a reviewed procedure.
func (w *Workspace) SaveSkill(name, description, instructions string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\") || strings.TrimSpace(description) == "" {
		return fmt.Errorf("skill requires a directory name and description")
	}
	header, err := yaml.Marshal(Skill{Name: name, Description: description})
	if err != nil {
		return err
	}
	content := "---\n" + string(header) + "---\n\n" + instructions + "\n"
	if len(content) > maxOutput {
		return fmt.Errorf("skill exceeds 64 KiB")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	root, err := os.OpenRoot(w.dir)
	if err != nil {
		return err
	}
	defer root.Close()
	dir := path.Join(".agents/skills", name)
	if err := root.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := root.OpenFile(path.Join(dir, "SKILL.md"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = file.WriteString(content)
	closeErr := file.Close()
	return errors.Join(err, closeErr)
}

func (w *Workspace) saveSkill(ctx context.Context, input map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	name, err := argument(input, "name")
	if err != nil {
		return "", err
	}
	description, err := argument(input, "description")
	if err != nil {
		return "", err
	}
	instructions, err := argument(input, "instructions")
	if err != nil {
		return "", err
	}
	if err := w.SaveSkill(name, description, instructions); err != nil {
		return "", err
	}
	return path.Join(".agents/skills", name, "SKILL.md"), nil
}
