//go:build unix

package attachments

import (
	"os"

	"golang.org/x/sys/unix"
)

func freeBytes(dir string) (uint64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(dir, &st); err != nil {
		return 0, err
	}
	return st.Bavail * uint64(st.Bsize), nil
}

// syncDir makes the rename durable. Errors are ignored: the file is already
// in place, and a failed directory sync only weakens crash durability.
func syncDir(dir string) {
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
}
