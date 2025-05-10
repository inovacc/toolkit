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

// Node represents a node in the file tree, typically a file or directory.
type Node struct {
	Name     string  `json:"name"`
	Children []*Node `json:"children,omitempty"`
}

// Tree is a filesystem tree generator that walks a directory structure
// and can output it in multiple formats: string tree, JSON, and Markdown.
type Tree struct {
	fs      afero.Fs
	root    *Node
	path    string
	exclude []string
}

// NewTree constructs a new Tree given a filesystem interface and root path.
// You can optionally pass directory/file names to exclude from traversal.
func NewTree(fs afero.Fs, path string, exclude ...string) *Tree {
	return &Tree{
		fs:      fs,
		path:    path,
		root:    &Node{Name: filepath.Base(path)},
		exclude: exclude,
	}
}

// MakeTree initiates the tree construction by walking through the filesystem
// starting from the root path.
func (t *Tree) MakeTree() error {
	if t.fs == nil {
		return errors.New("nil filesystem")
	}
	return t.buildNode(t.path, t.root)
}

// ToJSON returns the directory structure encoded in JSON format.
func (t *Tree) ToJSON() (string, error) {
	data, err := json.MarshalIndent(t.root, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToMarkdown returns the directory structure in a Markdown list format.
func (t *Tree) ToMarkdown() string {
	var b strings.Builder
	t.writeMarkdown(&b, t.root, "")
	return b.String()
}

// ToString returns the directory structure in a human-readable "tree" CLI-like format.
func (t *Tree) ToString() string {
	var b strings.Builder
	t.writeTreeFormat(&b, t.root, "", true)
	return b.String()
}

// writeTreeFormat recursively writes a visual tree format to the string builder.
func (t *Tree) writeTreeFormat(b *strings.Builder, node *Node, prefix string, isRoot bool) {
	if isRoot {
		b.WriteString(".\n")
	}

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
			t.writeTreeFormat(b, child, prefix+next, false)
		}
	}
}

// writeMarkdown recursively writes a Markdown representation of the tree.
func (t *Tree) writeMarkdown(b *strings.Builder, node *Node, prefix string) {
	_, _ = fmt.Fprintf(b, "%s- %s\n", prefix, node.Name)
	for _, child := range node.Children {
		t.writeMarkdown(b, child, "  "+prefix)
	}
}

// buildNode reads a directory and recursively builds nodes for its contents.
func (t *Tree) buildNode(path string, parent *Node) error {
	entries, err := afero.ReadDir(t.fs, path)
	if err != nil {
		return err
	}

	// Sort directories before files, and alphabetically within types
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if t.shouldExclude(entry.Name()) {
			continue
		}

		child := &Node{Name: entry.Name()}
		parent.Children = append(parent.Children, child)

		if entry.IsDir() {
			err := t.buildNode(filepath.Join(path, entry.Name()), child)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// shouldExclude checks if a filename matches the exclude list.
func (t *Tree) shouldExclude(name string) bool {
	for _, ex := range t.exclude {
		if ex == name {
			return true
		}
	}
	return false
}
