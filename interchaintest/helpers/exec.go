package helpers

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
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
	cli := node.DockerClient
	created, err := cli.ContainerExecCreate(ctx, node.Name(), container.ExecOptions{
		Cmd:          cmd,
		Env:          node.Chain.Config().Env,
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
// creating its directory. It copies a tar into the running container, so it
// needs no tools in the image (the simd image has no shell utilities).
func writeNodeFile(ctx context.Context, node *cosmos.ChainNode, content []byte, relPath string) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if dir := path.Dir(relPath); dir != "." {
		if err := tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: dir + "/", Mode: 0o755}); err != nil {
			return err
		}
	}
	if err := tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: relPath, Mode: 0o644, Size: int64(len(content))}); err != nil {
		return err
	}
	if _, err := tw.Write(content); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := node.DockerClient.CopyToContainer(ctx, node.Name(), node.HomeDir(), &buf, container.CopyToContainerOptions{}); err != nil {
		return fmt.Errorf("copy %s into %s: %w", relPath, node.Name(), err)
	}
	return nil
}
