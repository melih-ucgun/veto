package core

import (
	"context"
	"io"
)

// Transport is the interface for executing commands and managing files
// across different communication channels (local, SSH, etc.)
type Transport interface {
	io.Closer

	// Execute runs a command and returns its combined output
	Execute(ctx context.Context, cmd string) (string, error)

	// CopyFile sends a local file to the remote system
	CopyFile(ctx context.Context, localPath, remotePath string) error

	// DownloadFile retrieves a file from the remote system
	DownloadFile(ctx context.Context, remotePath, localPath string) error

	// GetFileSystem returns the FileSystem abstraction for this transport
	GetFileSystem() FileSystem

	// GetOS returns the operating system name (e.g., "linux", "darwin")
	GetOS(ctx context.Context) (string, error)
}
