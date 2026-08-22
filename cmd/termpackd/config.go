package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const defaultAddress = "127.0.0.1:19081"

type config struct {
	address   string
	database  string
	selfcheck bool
}

func parseConfig(args []string) (config, error) {
	set := flag.NewFlagSet("termpackd", flag.ContinueOnError)
	var cfg config
	set.StringVar(&cfg.address, "addr", "", "HTTP 监听地址")
	set.StringVar(&cfg.database, "db", "termpacks.db", "SQLite 数据库路径")
	set.BoolVar(&cfg.selfcheck, "selfcheck", false, "执行完整业务自检后退出")
	if err := set.Parse(args); err != nil {
		return cfg, err
	}
	if set.NArg() != 0 {
		return cfg, errors.New("不接受位置参数")
	}
	if cfg.address == "" {
		cfg.address = defaultAddress
		if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
			value, err := strconv.Atoi(port)
			if err != nil || value < 1024 || value > 65535 {
				return cfg, fmt.Errorf("PORT 必须是 1024 至 65535 的端口号")
			}
			cfg.address = net.JoinHostPort("127.0.0.1", port)
		}
	}
	if err := validateAddress(cfg.address); err != nil {
		return cfg, err
	}
	if strings.TrimSpace(cfg.database) == "" {
		return cfg, errors.New("数据库路径不能为空")
	}
	if cfg.selfcheck {
		cfg.database = ":memory:"
	}
	return cfg, nil
}

func validateAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("-addr 必须采用 host:port 格式: %w", err)
	}
	if strings.TrimSpace(host) == "" {
		return errors.New("-addr 必须明确指定主机地址")
	}
	value, err := strconv.Atoi(port)
	if err != nil || value < 1 || value > 65535 {
		return errors.New("-addr 端口必须在 1 至 65535 之间")
	}
	return nil
}
