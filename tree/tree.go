package tree

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/afero"
)

const (
	connector       = "├── "
	nextPrefix      = "│   "
	connectorChild  = "└── "
	nextPrefixChild = "    "
)

type OptsFn func(opts *Config)

type Config struct {
	exclude []string
}

func NewConfig(o ...OptsFn) *Config {
	cfg := &Config{}
	for _, opts := range o {
		opts(cfg)
	}
	return cfg
}

func WithExclude(data ...string) OptsFn {
	return func(opts *Config) {
		opts.exclude = append(opts.exclude, data...)
	}
}

type Node struct {
	Name     string  `json:"name"`
	Children []*Node `json:"children,omitempty"`
}

type Tree struct {
	cfg  *Config
	fs   afero.Fs
	root *Node
	path string
}

func NewTree(fs afero.Fs, path string, cfg *Config) *Tree {
	return &Tree{
		fs:   fs,
		path: path,
		root: &Node{Name: filepath.Base(path)}, // Display the base name for root, or "." if a path is "."
		cfg:  cfg,
	}
}

func (t *Tree) MakeTree() error {
	if t.fs == nil {
		return errors.New("nil filesystem")
	}
	// Adjust the root node name if the input path is "."
	if t.path == "." {
		t.root.Name = "."
	}
	return t.buildNode(t.path, t.root)
}

func (t *Tree) ToJSON() (string, error) {
	data, err := json.MarshalIndent(t.root, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (t *Tree) ToMarkdown() string {
	var b strings.Builder
	// For Markdown, usually we start with the root name without a prefix bullet
	_, _ = fmt.Fprintf(&b, "- %s\n", t.root.Name)
	for _, child := range t.root.Children {
		t.writeMarkdown(&b, child, "  ")
	}
	return b.String()
}

func (t *Tree) ToString() string {
	var b strings.Builder
	// Start with the root name for ToString as well.
	// The original writeTreeFormat started with ".\n" unconditionally for the root's children.
	// We'll print the root name first, then its children.
	b.WriteString(t.root.Name + "\n")
	t.writeTreeFormat(&b, t.root, "", false) // Pass false for isRoot initially as the root is already printed
	return b.String()
}

func (t *Tree) writeTreeFormat(b *strings.Builder, node *Node, prefix string, isRootAlreadyPrinted bool) {
	// isRootAlreadyPrinted is a bit of a misnomer now, it's more about whether we are processing children of the displayed root
	// If the node passed is the actual root, its name is printed before calling this.
	// This function now focuses on printing children.

	for i, child := range node.Children {
		isLast := i == len(node.Children)-1
		conn := connector
		next := nextPrefix

		if isLast {
			conn = connectorChild
			next = nextPrefixChild
		}

		_, _ = fmt.Fprintf(b, "%s%s%s\n", prefix, conn, child.Name)

		if len(child.Children) > 0 {
			// For children, the prefix logic is correct.
			// The isRoot flag (now isRootAlreadyPrinted) was originally for the ".\n" line, which is handled differently now.
			t.writeTreeFormat(b, child, fmt.Sprintf("%s%s", prefix, next), isRootAlreadyPrinted)
		}
	}
}

func (t *Tree) writeMarkdown(b *strings.Builder, node *Node, prefix string) {
	_, _ = fmt.Fprintf(b, "%s- %s\n", prefix, node.Name)
	for _, child := range node.Children {
		t.writeMarkdown(b, child, "  "+prefix)
	}
}

func (t *Tree) buildNode(currentPath string, parent *Node) error {
	entries, err := afero.ReadDir(t.fs, currentPath)
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir() // Directories first
		}
		return entries[i].Name() < entries[j].Name() // Then by name
	})

	for _, entry := range entries {
		// Check if the current entry (file or directory) should be excluded
		if t.isEntryExcluded(entry.Name()) {
			continue // Skip this entry entirely
		}

		child := &Node{Name: entry.Name()}
		parent.Children = append(parent.Children, child)

		// If it's a directory (and was not excluded above), then recurse
		if entry.IsDir() {
			err := t.buildNode(filepath.Join(currentPath, entry.Name()), child)
			if err != nil {
				return err // Propágate error
			}
		}
	}
	return nil
}

// there isEntryExcluded checks if the given name (file or directory) matches any of the configured exclude patterns.
func (t *Tree) isEntryExcluded(name string) bool {
	for _, pattern := range t.cfg.exclude {
		// Using HasPrefix as per the original implicit logic for directories.
		// This can be changed to exact match or glob matching if needed.
		if strings.HasPrefix(name, pattern) {
			return true
		}
	}
	return false
}
