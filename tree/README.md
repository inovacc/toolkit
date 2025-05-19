# Go Directory Tree Generator

A Go package for generating directory tree structures. It can represent the file system hierarchy as a formatted string, JSON, or Markdown. It also supports excluding specific files and directories from the tree based on name prefixes.

This package uses [`afero`](https://github.com/spf13/afero) for filesystem abstraction, allowing it to work seamlessly with the actual OS filesystem or in-memory filesystems for testing and other purposes.

## Features

* List directory and file structures are recursive.
* Exclude files and directories based on prefix patterns (e.g., `node_modules`, `.git`).
* Output in multiple formats:
    * Formatted string (similar to the Unix `tree` command).
    * JSON.
    * Markdown list.
* Filesystem agnostic due to `afero` integration.

## Installation

To use this package in your Go project, import it using its module path:

```go
import "github.com/inovacc/toolkit/tree" 