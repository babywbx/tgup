package files

import (
	"io/fs"
	"os"
)

// FS abstracts filesystem operations used by deterministic modules.
type FS interface {
	Stat(name string) (fs.FileInfo, error)
	Lstat(name string) (fs.FileInfo, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	Open(name string) (fs.File, error)
}

// OSFS implements FS using the local OS filesystem.
type OSFS struct{}

// Stat implements FS.
func (OSFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// Lstat implements FS.
func (OSFS) Lstat(name string) (fs.FileInfo, error) {
	return os.Lstat(name)
}

// ReadDir implements FS.
func (OSFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

// Open implements FS.
func (OSFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}
