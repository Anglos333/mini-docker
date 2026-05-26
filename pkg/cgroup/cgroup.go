package cgroup

import (
	"fmt"
	"os"
	"path"
	"strconv"
)

// EnableSubtree 自动化权限下放
func EnableSubtree() {
	subtreePath := "/sys/fs/cgroup/cgroup.subtree_control"
	if err := os.WriteFile(subtreePath, []byte("+memory"), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "[cgroup] 自动化开启内存限制失败: %v\n", err)
	}
}

// SetMemoryLimit 为容器创建专属 Cgroup 并限制内存
func SetMemoryLimit(shortID string, pid int) string {
	containerID := fmt.Sprintf("mini-docker-%s", shortID)
	cgroupPath := path.Join("/sys/fs/cgroup", containerID)
	os.Mkdir(cgroupPath, 0755)
	os.WriteFile(path.Join(cgroupPath, "memory.max"), []byte("104857600"), 0644)
	os.WriteFile(path.Join(cgroupPath, "cgroup.procs"), []byte(strconv.Itoa(pid)), 0644)
	return cgroupPath
}