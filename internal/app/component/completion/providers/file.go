package providers

import (
	"app/component/completion"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileProvider completes file and directory paths
type FileProvider struct{}

func NewFileProvider() *FileProvider { return &FileProvider{} }

// DirectoryProvider completes directory paths only
type DirectoryProvider struct{}

func NewDirectoryProvider() *DirectoryProvider { return &DirectoryProvider{} }

func (p *FileProvider) Fetch(ctx completion.Context) completion.Result {
	return fetchFileSystem(ctx.Word, false)
}

func (p *DirectoryProvider) Fetch(ctx completion.Context) completion.Result {
	return fetchFileSystem(ctx.Word, true)
}

func (p *FileProvider) ApplyTo(rawInput string, chosen string) string {
	return completion.ReplaceLastToken(rawInput, chosen)
}

func (p *DirectoryProvider) ApplyTo(rawInput string, chosen string) string {
	return completion.ReplaceLastToken(rawInput, chosen)
}

type fileEntry struct {
	// completion is the full absolute (quoted) path inserted into the line
	completion string

	// displayName is the bare basename shown in the hint bar
	displayName string
}

func fetchFileSystem(input string, dirsOnly bool) completion.Result {

	// Strip surrounding quotes the user may have typed
	input = strings.Trim(input, `"`)

	// Split current user input into directory and filename values
	searchDir, fileStart := splitPath(input)

	entries, err := os.ReadDir(searchDir)

	if err != nil {
		return completion.Result{}
	}

	sep := string(filepath.Separator)
	var matches []fileEntry

	// For directory completion also offer . and ..
	if dirsOnly {
		matches = append(matches, dotEntries(searchDir, fileStart, sep)...)
	}

	for _, e := range entries {
		if dirsOnly && !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		if !strings.HasPrefix(name, fileStart) {
			continue
		}

		abs, err := filepath.Abs(filepath.Join(searchDir, name))
		if err != nil {
			abs = filepath.Join(searchDir, name)
		}

		displayName := name
		if e.IsDir() {
			abs += sep
			displayName += sep
		}

		comp := abs
		if strings.Contains(comp, " ") {
			comp = `"` + comp + `"`
		}

		matches = append(matches, fileEntry{completion: comp, displayName: displayName})
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].displayName < matches[j].displayName
	})

	completions := make([]string, len(matches))
	displayNames := make([]string, len(matches))
	for i, m := range matches {
		completions[i] = m.completion
		displayNames[i] = m.displayName
	}
	return completion.Result{Completions: completions, DisplayNames: displayNames}
}

// splitPath splits a raw path string into (directory, filename).
func splitPath(path string) (dir string, base string) {
	if path == "" {
		return ".", ""
	}
	sep := string(filepath.Separator)
	if strings.HasSuffix(path, sep) {
		return path, ""
	}
	if strings.Contains(path, sep) {
		return filepath.Dir(path), filepath.Base(path)
	}
	return ".", path
}

// dotEntries produces entries for "." and ".." when completing directories.
func dotEntries(searchDir, fileStart, sep string) []fileEntry {
	var out []fileEntry
	candidates := []struct {
		match   string
		relPath string
	}{
		{".", searchDir},
		{"..", filepath.Join(searchDir, "..")},
	}
	for _, c := range candidates {
		if !strings.HasPrefix(c.match, fileStart) {
			continue
		}
		abs, _ := filepath.Abs(c.relPath)
		comp := abs + sep
		if strings.Contains(comp, " ") {
			comp = `"` + comp + `"`
		}
		out = append(out, fileEntry{completion: comp, displayName: c.match + sep})
	}
	return out
}
