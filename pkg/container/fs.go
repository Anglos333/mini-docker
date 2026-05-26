package container

import (
	"fmt"
	"os"
	"path"
	"syscall"
)

// SetupOverlayFS 创建联合文件系统并返回最终挂载点
func SetupOverlayFS(shortID string) (string, string, string) {
	overlayBase := "/home/Go/mini-docker/overlay"
	lowerDir := "/home/Go/mini-docker/rootfs"
	upperDir := path.Join(overlayBase, "upper-"+shortID)
	workDir := path.Join(overlayBase, "work-"+shortID)
	mergedDir := path.Join(overlayBase, "merged-"+shortID)

	os.MkdirAll(upperDir, 0755)
	os.MkdirAll(workDir, 0755)
	os.MkdirAll(mergedDir, 0755)

	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDir, upperDir, workDir)
	if err := syscall.Mount("overlay", mergedDir, "overlay", 0, opts); err != nil {
		fmt.Fprintf(os.Stderr, "[fs] Overlay Mount Error: %v\n", err)
	}
	return mergedDir, upperDir, workDir
}

// PivotRoot 执行跨次元地基替换
func PivotRoot(mergedDir string) {
	syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	syscall.Mount(mergedDir, mergedDir, "bind", syscall.MS_BIND|syscall.MS_REC, "")
	pivotDir := path.Join(mergedDir, ".pivot_root")
	os.Mkdir(pivotDir, 0777)
	syscall.PivotRoot(mergedDir, pivotDir)
	syscall.Chdir("/")
	syscall.Mount("proc", "/proc", "proc", uintptr(syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV), "")
	syscall.Unmount("/.pivot_root", syscall.MNT_DETACH)
	os.RemoveAll("/.pivot_root")
}