package transport

import (
	"context"
	"fmt"
	"runtime"

	"github.com/melih-ucgun/veto/internal/core"
)

// LocalTransport implements Transport for the local machine
type LocalTransport struct{}

func NewLocalTransport() *LocalTransport {
	return &LocalTransport{}
}

func (t *LocalTransport) Close() error {
	return nil
}

func (t *LocalTransport) Execute(ctx context.Context, cmd string) (string, error) {
	// For local execution, we use the global CommandRunner.
	// We wrap the command string in a shell to ensure compatibility with remote execution.
	out, err := core.RunCommand("sh", "-c", cmd)
	if err != nil {
		fmt.Printf("DEBUG(transport): Execute fail: cmd=[%s] err=[%v] out=[%s]\n", cmd, err, out)
	}
	return out, err
}

func (t *LocalTransport) CopyFile(ctx context.Context, localPath, remotePath string) error {
	// For local transport, localPath and remotePath are on the same machine.
	fs := &core.RealFS{}
	return core.CopyFile(fs, localPath, remotePath, 0644)
}

func (t *LocalTransport) DownloadFile(ctx context.Context, remotePath, localPath string) error {
	// Local download is just a copy.
	return t.CopyFile(ctx, remotePath, localPath)
}

func (t *LocalTransport) GetFileSystem() core.FileSystem {
	return &core.RealFS{}
}

func (t *LocalTransport) GetOS(ctx context.Context) (string, error) {
	return runtime.GOOS, nil
}
