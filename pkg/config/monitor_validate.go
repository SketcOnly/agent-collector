package config

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// Validate HTTP服务配置校验
func (h *ServerConfig) Validate() error {
	if err := valid.Struct(h); err != nil {
		return err
	}
	// 	校验Addr格式(必须是 ":port" 或 "ip:port")

	if h.Addr == "" {
		return errors.New("[ERROR] HTTP.Addr cannot be empty")
	}
	// 	用net包解析地址，验证格式合法性
	_, err := net.ResolveTCPAddr("tcp", h.Addr)
	if err != nil {
		return fmt.Errorf("[ERROR] HTTP.Addr format invalid (expected: :port or ip:port), got %s: %w", h.Addr, err)
	}

	return nil
}

func (m *MonitorConfig) Validate() error {
	if err := valid.Struct(m); err != nil {
		return err
	}
	if m.Interval < time.Second || m.Interval > 3600*time.Second {
		return fmt.Errorf("monitor,interval must be between 1 and 3600 seconds, got %s", m.Interval)
	}
	// CollectorConfig 加盐

	if err := m.Collectors.validate(); err != nil {
		return err
	}
	return nil
}

// 校验至少启用一个采集，否则没有意义
func (col *CollectorConfig) validate() error {
	if err := valid.Struct(col); err != nil {
		return err
	}
	// 	校验至少启用一个采集器，否则没有意义
	if !col.Proc.Enable && !col.Sys.Enable && !col.Cgroup.Enable && !col.Container.Enable {
		return fmt.Errorf("at least one collector must be enabled (proc/sys/cgroup/container)")
	}
	//	 sys 采集器校验
	if err := col.Sys.Validate(); err != nil {
		return err
	}

	return nil
}

// Validate 忽略列表不能包含空字符串
// 忽略的磁盘必须看起来像 "/dev/**"
// 忽略的网络接口格式必须合法（不能有空格、不能是奇怪字符）
// 重复项检测（避免配置写错）
// sys 未启用时不校验
// 不会误判，也不会过于严格导致难用。

func (col *SysDataSourceConfig) Validate() error {
	if err := valid.Struct(col); err != nil {
		return err
	}

	// 如果 sys未启用时不参与校验
	if !col.Enable {
		return nil
	}

	// 	校验 IgnoreDisks
	seenDisk := map[string]bool{}
	for _, d := range col.IgnoreDisks {

		if strings.TrimSpace(d) == "" {
			return fmt.Errorf("sys.ignore_disks cannot contain empty string")
		}
		//	要求必须为/dev/xxx
		if seenDisk[d] {
			return fmt.Errorf("sys.ignore_disks contains duplicate disk name: %s", d)
		}

		seenDisk[d] = true
	}

	// 校验IgnoreNetworks
	seeniface := map[string]bool{}
	for _, iface := range col.IgnoreNetworks {
		if strings.TrimSpace(iface) == "" {
			return fmt.Errorf("sys.ignore_networks cannot contain empty string")
		}

		//	网络接口名称要求：不能有空格，不能包含奇怪字符
		// 通常linux 接口名如 etho,enp0sa,lo,docker0..
		if strings.ContainsAny(iface, "\t\r\n") {
			return fmt.Errorf("sys.ignore_networks: interface %q contains whitespace", iface)
		}
		if strings.ContainsAny(iface, "/\\") {
			return fmt.Errorf("sys.ignore_networks: interface %q must not contain '/' or '\\\\", iface)
		}

		// 重复项检查
		if seeniface[iface] {
			return fmt.Errorf("sys.ignore_networks duplicated entry: %q", iface)
		}
		seeniface[iface] = true
	}
	return nil
}
