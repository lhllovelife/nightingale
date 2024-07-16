package conf

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/ccfos/nightingale/v6/alert/aconf"
	"github.com/ccfos/nightingale/v6/center/cconf"
	"github.com/ccfos/nightingale/v6/pkg/cfg"
	"github.com/ccfos/nightingale/v6/pkg/httpx"
	"github.com/ccfos/nightingale/v6/pkg/logx"
	"github.com/ccfos/nightingale/v6/pkg/ormx"
	"github.com/ccfos/nightingale/v6/pushgw/pconf"
	"github.com/ccfos/nightingale/v6/storage"
)

type ConfigType struct {
	Global    GlobalConfig
	Log       logx.Config
	HTTP      httpx.Config
	DB        ormx.DBConfig
	Redis     storage.RedisConfig
	CenterApi CenterApi

	Pushgw pconf.Pushgw
	Alert  aconf.Alert
	Center cconf.Center
	Ibex   Ibex
}

type CenterApi struct {
	Addrs         []string
	BasicAuthUser string
	BasicAuthPass string
	Timeout       int64
}

type GlobalConfig struct {
	RunMode string
}

type Ibex struct {
	Enable    bool
	RPCListen string
	Output    Output
}

type Output struct {
	ComeFrom string
	AgtdPort int
}

func InitConfig(configDir, cryptoKey string) (*ConfigType, error) {
	var config = new(ConfigType) // 分配一块ConfigType类型的内存，初始化值为对应的零值，返回新分配内存的地址

	// 从指定目录加载配置文件内容写到config地址对应的内存
	if err := cfg.LoadConfigByDir(configDir, config); err != nil {
		return nil, fmt.Errorf("failed to load configs of directory: %s error: %s", configDir, err)
	}

	config.Pushgw.PreCheck()         // Pushgw配置参数前置检查
	config.Alert.PreCheck(configDir) // Alert配置参数前置检查
	config.Center.PreCheck()         // Center配置参数前置检查

	err := decryptConfig(config, cryptoKey)
	if err != nil {
		return nil, err
	}

	// Alert心跳 IP配置检查
	if config.Alert.Heartbeat.IP == "" {
		// auto detect
		config.Alert.Heartbeat.IP = fmt.Sprint(GetOutboundIP())
		if config.Alert.Heartbeat.IP == "" {
			hostname, err := os.Hostname()
			if err != nil {
				fmt.Println("failed to get hostname:", err)
				os.Exit(1)
			}

			if strings.Contains(hostname, "localhost") {
				fmt.Println("Warning! hostname contains substring localhost, setting a more unique hostname is recommended")
			}

			config.Alert.Heartbeat.IP = hostname
		}
	}

	config.Alert.Heartbeat.Endpoint = fmt.Sprintf("%s:%d", config.Alert.Heartbeat.IP, config.HTTP.Port)

	return config, nil
}

// 从连接中获取本地地址并返回IP部分
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "223.5.5.5:80")
	if err != nil {
		fmt.Println("auto get outbound ip fail:", err)
		return []byte{}
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
