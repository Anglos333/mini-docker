# mini-docker 🚀

A lightweight, from-scratch container runtime implemented in Go, designed to fully unveil the core mechanics of modern containerization technology (Docker/runc).

`mini-docker` 是一个基于 **Go 语言和 Linux 内核底层特性** 从零实现的轻量级沙箱容器运行时。项目摒弃了繁琐的工业包装，用高内聚、低耦合的模块化架构，完整闭环了现代容器底层的核心隔离、硬资源配额、分层文件系统及网络自适应编排链路。

---

## 🌟 Core Features / 核心特性

* 🔒 **Advanced OS Isolation (高级系统级隔离)**
  * 利用 Linux **Namespaces** (`CLONE_NEWUTS`, `CLONE_NEWPID`, `CLONE_NEWNS`, `CLONE_NEWNET`) 实现了独立的主机名、进程树、挂载点以及网络协议栈隔离。
  * 弃用简陋的 `chroot`，底层硬核调用 `PivotRoot` 配合 `MS_PRIVATE` 挂载声明，实现容器与宿主机根文件系统的物理级截断，彻底焊死 **Jail Escape (越狱攻击)** 的大门。
* ⚡ **Automated High-Concurrency Networking (自动化网络并发)**
  * 运行时自研“网络无感编排机制”，父进程动态创建 **Veth Pair 虚拟网线**，利用 **PID 跨空间穿墙术** 将网线精准投递至隔离房间。
  * 容器内部触发“整容魔术”，统一将凌乱的随机网卡名修改为标准 `eth0`，配合纳秒级动态 IPAM 分配，彻底击碎多容器并发打架的魔咒。
* 🌍 **Zero-Dependency Public Gateway (环境自适应出海网关)**
  * 运行时具备宿主机网络栈全局感知能力，启动时**自动探测并初始化虚拟交换机 (Bridge br0)、绑定子网、下发全局 iptables NAT (MASQUERADE) 易容术规则**，赋予沙箱容器开箱即用的公网直通能力。
* 💾 **Modern Copy-on-Write Storage (镜像分层与“阅后即焚”)**
  * 深度集成 Linux **OverlayFS** 联合文件系统架构。将基础 rootfs 设为绝对只读的镜像层 (`lowerdir`)，并发容器动态挂载独立的读写层 (`upperdir`)。
  * 完美设计容器退出时的 `MNT_DETACH` 懒惰卸载与级联清理机制，实现容器内脏数据全自动“阅后即焚”，永不污染宿主机母体。
* 🚀 **Daemon Mode & Lifecycle Management (常驻常驻与状态落盘)**
  * 巧妙利用 Linux **Pipe (管道同步发令枪)** 编排父子进程初始化流，确保 Cgroups v2 内存配额手铐在子进程 Shell 启动的首纳秒内精准闭环。
  * 支持 **`-d` (Detach) 后台常驻模式**，通过解耦文件描述符重定向，并将容器状态字典（Metadata）持久化序列化至 `/var/run/mini-docker/` 目录。

---

## 🏗️ Architecture Diagram / 技术拓扑架构

```text
 宿主机工作空间 (Host Sandbox Workspace)
  │
  ├── [父进程 Parent Process] (参数解析、Cgroups v2 内存限额、网桥 NAT 全自动开闸)
  │    │
  │    ├── 1. 动态派生专属虚拟网线 (vh-xxxx <---> vc-xxxx)
  │    ├── 2. 隔空投递：通过子进程 PID 将 vc-xxxx 强行塞入隔离房间
  │    └── 3. 扣动管道同步发令枪 ("ok")
  │
  └── [子进程 Child Process] (罩上 Namespace 魔法保护伞)
       │
       ├── 1. 管道解封醒来 & 立即执行网卡整容：vc-xxxx ──> 标准 eth0
       ├── 2. 挂载 OverlayFS 联合文件系统 (将合并层 merged 作为容器最终根目录)
       ├── 3. 乾坤大挪移：执行 PivotRoot 彻底替换地基 & Chdir("/") & 挂载专属 /proc
       └── 4. 真正拉起容器 Shell (处于绝对完美隔离、限额、通网的 BusyBox 沙箱空间)

```

---

## 📂 Directory Structure / 工业级目录规范


```text
mini-docker/
├── main.go               # 项目全局唯一总入口
├── go.mod
├── go.sum
└── pkg/                  # 底层硬核内核组件箱 (不包含任何命令业务)
    ├── cgroup/
    │   └── cgroup.go     # 负责 Cgroup v2 权限下放、目录创建、100MB 内存硬限制
    ├── container/
    │   └── fs.go         # 负责 OverlayFS 联合挂载、PivotRoot 地基置换、/proc 隔离
    └── network/
        └── network.go    # 负责 宿主机网桥自愈、iptables NAT 下发、Veth 穿墙整容
└── cmd/                  # 业务指挥部 (负责 Cobra 命令行解析与多模式常驻编排)
    ├── root.go           # 注册全局 Flag 标志位
    ├── run.go            # 父进程核心业务流 (配置宿主机、同步管道)
    └── child.go          # 子进程核心业务流 (配置容器内部、降维打击入沙箱)

```

---

## 🛠️ Quick Start / 快速通关指南

### Pre-requirements / 前置环境

确保你的运行环境为 Linux (Ubuntu 22.04+ 或启用 Cgroups v2 的 WSL2 虚拟机)，并且拥有 `root` 上帝权限。

```bash
# 1. 克隆准工业级完全体项目
git clone [https://github.com/yourusername/mini-docker.git](https://github.com/yourusername/mini-docker.git)
cd mini-docker

# 2. 在项目根目录下准备一个干净的 BusyBox rootfs 作为基础镜像层
mkdir -p rootfs
# (请自行解压或下载 busybox rootfs 放入 ./rootfs 目录下)

```

### 运行模式一：前台直通模式 (Foreground Mode)

```bash
sudo go run main.go run
# 输出实况：
# [parent] start
# [network] br0 不存在，全自动初始化网络枢纽...
# [parent] 容器成功拉起! ID: 52ba07, PID: 14820, IP: 192.168.200.117/24
# [parent] cgroups memory limit configured!
# [child] got setup signal, starting container...
# / # 

```

### 运行模式二：后台常驻模式 (Background Mode)

```bash
sudo go run main.go run -d
# 父进程在配置完 Cgroup 与自动化布线后会瞬间优雅退出，将容器托管给系统的 init 进程，常驻后台。

```

---

### 📸 运行实况展示 (Operation Screenshots)

| 1. Auto-Bridge & Run (一键开闸启动) | 2. UTS Hostname Isolation (系统级空间隔离) |
| --- | --- |
|  |  |
|<img width="415" height="160" alt="{303A1E22-1697-40AB-9C01-0F5026BF5A08}" src="https://github.com/user-attachments/assets/f42a2b08-b324-42e9-8bf8-a3f81add0195" />| <img width="415" height="160" alt="{FADD9A49-360F-4AA7-B634-A1AE2641EFD6}" src="https://github.com/user-attachments/assets/06877cf1-58b7-4d4b-83ef-0900d6d3ce52" />|

| 3. Public Internet Access (秒级跨次元通网) | 4. OverlayFS Copy-on-Write (阅后即焚测试) |
| --- | --- |
|  |  |
| <img width="415" height="160" alt="{8383ED6E-5C8E-43A1-BC7C-EFDB723C876A}" src="https://github.com/user-attachments/assets/fa3665f9-537f-4d0b-be70-9ea06822f44a" />|<img width="415" height="160" alt="{24056CEF-6A38-4553-B860-D19D6F201325}" src="https://github.com/user-attachments/assets/9003ed86-4583-48dc-9f37-b2e46e926069" />|

---

## 🛑 Technical Challenges & Pitfalls (技术难点与硬核踩坑复盘)

### 1. PivotRoot 与 Mount Namespace 传播污染 (Mount Propagation)

* **痛点**：在早期的单体实现中，执行 `syscall.PivotRoot` 经常无故抛出 `EINVAL` 错误卡死，甚至发生容器内挂载行为辐射导致宿主机根目录被意外解挂的惨剧。
* **攻克方案**：深入解析 Linux VFS 挂载树后发现，现代 Linux 系统（如 Ubuntu/WSL2）的默认挂载传播属性是 `shared`（共享）。为此，在执行 `PivotRoot` 之前，必须前置强制执行 `syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")` 将容器内部的挂载空间紧急声明为 `MS_PRIVATE`（私有隔离），彻底斩断时空连接，确保宿主机地基安全。

### 2. Cgroups v2 内存控制器自动下放时的权限陷阱

* **痛点**：在 WSL2/Linux 的重启或非原生环境中，父进程向 `memory.max` 写入限制配额时频繁遭遇 `Permission Denied` 拦截。
* **攻克方案**：Cgroups v2 引入了严格的“父子审判机制”。子目录想要开启内存控制器，其直属父目录必须提前放权。项目在父进程初始化阶段，自动前置改写宿主机 `/sys/fs/cgroup/cgroup.subtree_control` 强行写入 `+memory` 标志位。这一自动化设计彻底脱离了对宿主机手工预配置的强依赖，达成了“解压即用”的准工业级交付标准。

### 3. Veth Pair 跨空间投递导致的设备隐形与无感整容

* **痛点**：多容器并发运行时，写死虚拟网卡名字会导致宿主机内核网络设备秒级炸裂。
* **攻克方案**：项目采用“动态纳秒时间戳十六进制截取”技术，动态派生独一无二的网卡头（如 `vh-a1b2c3` 与 `vc-a1b2c3`）。利用 Linux “通过 PID 代表网络空间” 的特性投递至隔离域。最精妙的是，在容器文件系统被 `PivotRoot` 彻底截断前的“临界瞬间”，在容器空间内执行 `ip link set vc-xxxx name eth0`，将凌乱的随机网卡统一重构为标准的 `eth0`，实现了容器内部网络的无感封装。

### 4. Linux 内核三层转发（IP Forwarding）与宿主机拦截墙

* **痛点**：多容器并发网络拓扑搭建完成后，容器内成功分配 IP 且默认路由无误，但向外网发送报文时遭遇 100% 丢包，数据包在跨越虚拟网桥至新宿主机物理网卡时神秘蒸发。
* **攻克方案**：深入 Linux 内核网络栈排查，发现 WSL2 系统默认关闭了虚拟网卡间的转发总闸（`net.ipv4.ip_forward = 0`）。通过在宿主机前置自动下发 `sysctl -w net.ipv4.ip_forward=1` 强行拉开内核三层转发总闸，使数据包得以顺畅流向外部网络。该踩坑复盘深化了对 Linux VFS 网络报文转发及 Netfilter 过滤流向的理解。

---

## 📅 Project Roadmap / 进化旅程

* [x] Namespaces & UTS Hostname Isolation (多维空间与主机名隔离)
* [x] PivotRoot & Proc Filesystem Cutoff (地基截断与系统调用重组)
* [x] Cgroups v2 Automatic Resource Limitation (资源自动配额手铐)
* [x] Veth-Pair + Bridge Concurrent Networking (多容器并发互通网络)
* [x] Fully Automated Bridge & IPTables Initialization (宿主机网络全自动开闸)
* [x] OverlayFS Layered Image System (镜像分层及全自动清洁)
* [x] Detach Mode & Metadata Persistence (后台常驻与状态字典落盘)
* [ ] Implement `mini-docker ps` CLI command (支持容器活跃列表查询)
* [ ] Multi-Layered Image Tar pull & untar pipeline (对接标准 Tar 包镜像系统)
* [ ] Seccomp Syscall Whitelist Filtering (引入安全防护层白名单)
