package importer

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"time"
)

func runHook(hookCmd, owner, mountPath string, res Result) (string, error) {
	if hookCmd == "" {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", hookCmd)
	cmd.Env = append(cmd.Environ(),
		"CARDIMPORTD_OWNER="+owner,
		"CARDIMPORTD_MOUNT="+mountPath,
		"CARDIMPORTD_FILES_IMPORTED="+strconv.Itoa(res.Imported),
		"CARDIMPORTD_FILES_SKIPPED="+strconv.Itoa(res.Skipped),
		"CARDIMPORTD_FILES_FAILED="+strconv.Itoa(res.Failed),
		"CARDIMPORTD_BYTES_COPIED="+strconv.FormatInt(res.BytesCopied, 10),
	)
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		slog.Debug("post-import hook output", "output", string(out))
	}
	if err != nil {
		return "", fmt.Errorf("post-import hook: %w", err)
	}
	return string(out), nil
}
