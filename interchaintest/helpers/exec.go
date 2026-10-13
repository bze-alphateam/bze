package helpers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/docker/docker/api/types/container"
	"github.com/moby/moby/pkg/stdcopy"
)

// The helpers run every bzed CLI command inside the node's running container
// (docker exec), the way an operator would on the node. interchaintest's
// ChainNode.Exec starts a fresh container per command instead, which costs
// about half a second each; the suites send thousands of commands.

// execNode runs cmd in the node's container with the chain's environment. A
// non-zero exit returns an error worded like interchaintest's, "exit code N:
// <stdout> <stderr>", which the rejection and simulation parsing relies on.
func execNode(ctx context.Context, node *cosmos.ChainNode, cmd []string) ([]byte, []byte, error) {
	return execNodeStdin(ctx, node, cmd, nil)
}

// execNodeStdin is execNode with stdin fed from stdin (nil: no stdin).
func execNodeStdin(ctx context.Context, node *cosmos.ChainNode, cmd []string, stdin []byte) ([]byte, []byte, error) {
	cli := node.DockerClient
	created, err := cli.ContainerExecCreate(ctx, node.Name(), container.ExecOptions{
		Cmd:          cmd,
		Env:          node.Chain.Config().Env,
		AttachStdin:  stdin != nil,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("exec %s in %s: %w", strings.Join(cmd, " "), node.Name(), err)
	}
	attached, err := cli.ContainerExecAttach(ctx, created.ID, container.ExecAttachOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("attach exec %s in %s: %w", strings.Join(cmd, " "), node.Name(), err)
	}
	defer attached.Close()

	if stdin != nil {
		if _, err := io.Copy(attached.Conn, bytes.NewReader(stdin)); err != nil {
			return nil, nil, fmt.Errorf("write stdin of %s: %w", strings.Join(cmd, " "), err)
		}
		if err := attached.CloseWrite(); err != nil {
			return nil, nil, fmt.Errorf("close stdin of %s: %w", strings.Join(cmd, " "), err)
		}
	}

	var stdout, stderr bytes.Buffer
	copied := make(chan error, 1)
	go func() {
		_, err := stdcopy.StdCopy(&stdout, &stderr, attached.Reader)
		copied <- err
	}()
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case err := <-copied:
		if err != nil {
			return nil, nil, fmt.Errorf("read output of %s: %w", strings.Join(cmd, " "), err)
		}
	}

	inspect, err := cli.ContainerExecInspect(ctx, created.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("inspect exec %s: %w", strings.Join(cmd, " "), err)
	}
	if inspect.ExitCode != 0 {
		return nil, nil, fmt.Errorf("exit code %d: %s", inspect.ExitCode, strings.Join([]string{stdout.String(), stderr.String()}, " "))
	}
	return stdout.Bytes(), stderr.Bytes(), nil
}

// writeNodeFile writes content to relPath under the node's home directory,
// creating its directory, from inside the node's container.
func writeNodeFile(ctx context.Context, node *cosmos.ChainNode, content []byte, relPath string) error {
	dst := path.Join(node.HomeDir(), relPath)
	_, _, err := execNodeStdin(ctx, node, []string{"sh", "-c", `mkdir -p "$(dirname "$1")" && cat > "$1"`, "sh", dst}, content)
	return err
}
