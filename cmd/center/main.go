package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ccfos/nightingale/v6/center"
	"github.com/ccfos/nightingale/v6/pkg/osx"
	"github.com/ccfos/nightingale/v6/pkg/version"

	"github.com/toolkits/pkg/net/tcpx"
	"github.com/toolkits/pkg/runner"
)

var (
	showVersion = flag.Bool("version", false, "Show version.")
	configDir   = flag.String("configs", osx.GetEnv("N9E_CONFIGS", "etc"), "Specify configuration directory.(env:N9E_CONFIGS)")
	cryptoKey   = flag.String("crypto-key", "", "Specify the secret key for configuration file field encryption.")
)

func main() {
	flag.Parse() // 解析命令行参数

	if *showVersion { // 输出版本信息
		fmt.Println(version.Version)
		os.Exit(0) // 终止当前程序的执行并返回给定的状态码 (0 表示正常退出)
	}

	printEnv()

	tcpx.WaitHosts() // 等待外部服务或主机就绪 (主机信息通过环境变量获取)

	cleanFunc, err := center.Initialize(*configDir, *cryptoKey) // 初始化（如果初始化过程出现err, 终止程序）
	if err != nil {
		log.Fatalln("failed to initialize:", err)
	}

	/**
	使用系统信号实现程序的优雅退出
	*/
	code := 1
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

EXIT:
	for {
		sig := <-sc
		fmt.Println("received signal:", sig.String())
		switch sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			code = 0
			break EXIT
		case syscall.SIGHUP:
			// reload configuration?
		default:
			break EXIT
		}
	}

	// 执行收尾工作
	cleanFunc()
	fmt.Println("process exited")
	os.Exit(code)
}

func printEnv() {
	/* runner.Init
	1. 自动的根据CGROUP值识别容器的 CPU quota，并自动设置 GOMAXPROCS 线程数量
	2. 设置日志格式为日期、时间和短文件名
	3. 设置全局变量 Hostname 和 Cwd
	    - Hostname 主机名称(uname -n)
	    - Cwd 当前工作目录
	4. 设置随机数种子
	*/
	runner.Init()
	fmt.Println("runner.cwd:", runner.Cwd)
	fmt.Println("runner.hostname:", runner.Hostname)
	fmt.Println("runner.fd_limits:", runner.FdLimits())
	fmt.Println("runner.vm_limits:", runner.VMLimits())
}
