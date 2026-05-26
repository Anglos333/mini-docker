package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	detachMode bool // 🌟 用于接收 -d 参数
)

// 🌟 容器元数据结构体，用于 mini-docker ps
type ContainerInfo struct {
	ID          string    `json:"id"`
	PID         int       `json:"pid"`
	IP          string    `json:"ip"`
	CreatedTime time.Time `json:"created_time"`
}

var rootCmd = &cobra.Command{
	Use: "mini-docker",
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a container",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "[parent] start")

		// 1. 自动化权限下放 (Cgroup v2)
		subtreePath := "/sys/fs/cgroup/cgroup.subtree_control"
		if err := os.WriteFile(subtreePath, []byte("+memory"), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "[parent] 自动化开启 Cgroup 内存限制失败: %v\n", err)
			return
		}

		// 2. 🌟 自动化创建网桥 br0 和配置 iptables NAT 规则
		if err := exec.Command("ip", "link", "show", "br0").Run(); err != nil {
			fmt.Fprintln(os.Stderr, "[parent] br0 不存在，正在全自动初始化网络枢纽...")
			exec.Command("ip", "link", "add", "br0", "type", "bridge").Run()
			exec.Command("ip", "addr", "add", "192.168.200.1/24", "dev", "br0").Run()
			exec.Command("ip", "link", "set", "br0", "up").Run()
			// 自动配置宿主机 NAT 易容术规则
			exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", "192.168.200.0/24", "-o", "eth0", "-j", "MASQUERADE").Run()
		}

		// 3. 生成专属 ID 与动态网络参数
		nanoStr := fmt.Sprintf("%x", time.Now().UnixNano())
		shortID := nanoStr[len(nanoStr)-6:]
		vethHost := "vh-" + shortID
		vethChild := "vc-" + shortID
		ipLastOctet := int(time.Now().UnixNano()%250) + 2
		containerIP := fmt.Sprintf("192.168.200.%d/24", ipLastOctet)
		gatewayIP := "192.168.200.1"

		// 4. 🌟 打卡点二：OverlayFS 动态挂载（镜像不污染核心）
		overlayBase := "/home/Go/mini-docker/overlay"
		lowerDir := "/home/Go/mini-docker/rootfs" // 只读镜像层
		upperDir := path.Join(overlayBase, "upper-"+shortID)
		workDir := path.Join(overlayBase, "work-"+shortID)
		mergedDir := path.Join(overlayBase, "merged-"+shortID) // 最终容器根目录

		os.MkdirAll(upperDir, 0755)
		os.MkdirAll(workDir, 0755)
		os.MkdirAll(mergedDir, 0755)

		opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDir, upperDir, workDir)
		if err := syscall.Mount("overlay", mergedDir, "overlay", 0, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Overlay Mount Error: %v\n", err)
			return
		}

		// 创建管道同步
		readPipe, writePipe, err := os.Pipe()
		if err != nil {
			return
		}

		// 将 mergedDir 传给子进程作为最终的真实 rootfs
		command := exec.Command("/proc/self/exe", "child", containerIP, vethChild, gatewayIP, mergedDir)
		command.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET,
		}

		// 🌟 打卡点三：-d 后台运行重定向
		if detachMode {
			// 如果是后台运行，断开标准输入，将输出重定向到黑洞或日志文件，防止阻塞宿主机终端
			command.Stdout = nil
			command.Stderr = nil
			command.Stdin = nil
		} else {
			command.Stdin = os.Stdin
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
		}
		command.ExtraFiles = []*os.File{readPipe}

		if err := command.Start(); err != nil {
			return
		}
		readPipe.Close()
		pid := command.Process.Pid
		fmt.Fprintf(os.Stderr, "[parent] 容器成功拉起! ID: %s, PID: %d, IP: %s\n", shortID, pid, containerIP)

		// 5. 🌟 打卡点三：持久化存储容器 Metadata (用于随后的 ps 功能)
		metaDir := "/var/run/mini-docker"
		os.MkdirAll(metaDir, 0755)
		metaFilePath := path.Join(metaDir, shortID+".json")
		metaInfo := ContainerInfo{ID: shortID, PID: pid, IP: containerIP, CreatedTime: time.Now()}
		metaBytes, _ := json.Marshal(metaInfo)
		os.WriteFile(metaFilePath, metaBytes, 0644)

		// 6. Cgroups v2 内存限制
		containerID := fmt.Sprintf("mini-docker-%s", shortID)
		cgroupMemoryPath := path.Join("/sys/fs/cgroup", containerID)
		os.Mkdir(cgroupMemoryPath, 0755)
		os.WriteFile(path.Join(cgroupMemoryPath, "memory.max"), []byte("104857600"), 0644)
		os.WriteFile(path.Join(cgroupMemoryPath, "cgroup.procs"), []byte(strconv.Itoa(pid)), 0644)

		// 7. 父进程全自动网络接线
		exec.Command("ip", "link", "add", vethHost, "type", "veth", "peer", "name", vethChild).Run()
		exec.Command("ip", "link", "set", vethHost, "master", "br0").Run()
		exec.Command("ip", "link", "set", vethHost, "up").Run()
		exec.Command("ip", "link", "set", vethChild, "netns", strconv.Itoa(pid)).Run()

		// 扣动扳机，放行子进程
		writePipe.Write([]byte("ok"))
		writePipe.Close()

		// 🌟 如果是前台模式，父进程坐下看戏，并在退出时全自动打扫 Overlay 残留
		if !detachMode {
			command.Wait()
			os.Remove(metaFilePath)
			os.Remove(cgroupMemoryPath)
			syscall.Unmount(mergedDir, syscall.MNT_DETACH)
			os.RemoveAll(mergedDir)
			os.RemoveAll(upperDir)
			os.RemoveAll(workDir)
			fmt.Fprintln(os.Stderr, "[parent] 容器已安全退出，Overlay 写层已自动销毁清洁！")
		}
	},
}

var childCmd = &cobra.Command{
	Use:   "child",
	Short: "Child container process",
	Run: func(cmd *cobra.Command, args []string) {
		containerIP := args[0]
		vethChild := args[1]
		gatewayIP := args[2]
		mergedDir := args[3] // 🌟 接收来自父进程的合并挂载目录

		readPipe := os.NewFile(3, "pipe")
		msg := make([]byte, 2)
		readPipe.Read(msg)
		readPipe.Close()

		// Hostname 隔离生效
		syscall.Sethostname([]byte("mini-docker-container"))

		// 容器内部网络整容与配速
		exec.Command("ip", "link", "set", vethChild, "name", "eth0").Run()
		exec.Command("ip", "addr", "add", containerIP, "dev", "eth0").Run()
		exec.Command("ip", "link", "set", "eth0", "up").Run()
		exec.Command("ip", "link", "set", "lo", "up").Run()
		exec.Command("ip", "route", "add", "default", "via", gatewayIP).Run()

		// 🌟 使用 OverlayFS 的 mergedDir 进行强隔离 PivotRoot 
		syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
		syscall.Mount(mergedDir, mergedDir, "bind", syscall.MS_BIND|syscall.MS_REC, "")
		pivotDir := path.Join(mergedDir, ".pivot_root")
		os.Mkdir(pivotDir, 0777)
		syscall.PivotRoot(mergedDir, pivotDir)
		syscall.Chdir("/")
		syscall.Mount("proc", "/proc", "proc", uintptr(syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV), "")
		syscall.Unmount("/.pivot_root", syscall.MNT_DETACH)
		os.RemoveAll("/.pivot_root")

		command := exec.Command("/bin/sh", "-i")
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Run()
	},
}

func Execute() {
	// 🌟 注册后台运行的 -d 核心 Flag
	runCmd.Flags().BoolVarP(&detachMode, "detach", "d", false, "Run container in background")
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(childCmd)
	rootCmd.Execute()
}