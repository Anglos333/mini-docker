package network

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// InitBridge 自动化检查并创建宿主机网桥及 NAT
func InitBridge() {
	if err := exec.Command("ip", "link", "show", "br0").Run(); err != nil {
		fmt.Fprintln(os.Stderr, "[network] br0 不存在，全自动初始化网络枢纽...")
		exec.Command("ip", "link", "add", "br0", "type", "bridge").Run()
		exec.Command("ip", "addr", "add", "192.168.200.1/24", "dev", "br0").Run()
		exec.Command("ip", "link", "set", "br0", "up").Run()
		exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", "192.168.200.0/24", "-o", "eth0", "-j", "MASQUERADE").Run()
	}
}

// SetupHostVeth 在宿主机分配并接通虚拟网线
func SetupHostVeth(vethHost, vethChild string, pid int) {
	exec.Command("ip", "link", "add", vethHost, "type", "veth", "peer", "name", vethChild).Run()
	exec.Command("ip", "link", "set", vethHost, "master", "br0").Run()
	exec.Command("ip", "link", "set", vethHost, "up").Run()
	exec.Command("ip", "link", "set", vethChild, "netns", strconv.Itoa(pid)).Run()
}

// ConfigContainerNetwork 在容器内部进行网络整容和通电
func ConfigContainerNetwork(vethChild, containerIP, gatewayIP string) {
	exec.Command("ip", "link", "set", vethChild, "name", "eth0").Run()
	exec.Command("ip", "addr", "add", containerIP, "dev", "eth0").Run()
	exec.Command("ip", "link", "set", "eth0", "up").Run()
	exec.Command("ip", "link", "set", "lo", "up").Run()
	exec.Command("ip", "route", "add", "default", "via", gatewayIP).Run()
}