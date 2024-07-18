package center

import (
	"context"
	"fmt"

	"github.com/ccfos/nightingale/v6/alert"
	"github.com/ccfos/nightingale/v6/alert/astats"
	"github.com/ccfos/nightingale/v6/alert/process"
	alertrt "github.com/ccfos/nightingale/v6/alert/router"
	"github.com/ccfos/nightingale/v6/center/cconf"
	"github.com/ccfos/nightingale/v6/center/cconf/rsa"
	"github.com/ccfos/nightingale/v6/center/cstats"
	"github.com/ccfos/nightingale/v6/center/integration"
	"github.com/ccfos/nightingale/v6/center/metas"
	centerrt "github.com/ccfos/nightingale/v6/center/router"
	"github.com/ccfos/nightingale/v6/center/sso"
	"github.com/ccfos/nightingale/v6/conf"
	"github.com/ccfos/nightingale/v6/dumper"
	"github.com/ccfos/nightingale/v6/memsto"
	"github.com/ccfos/nightingale/v6/models"
	"github.com/ccfos/nightingale/v6/models/migrate"
	"github.com/ccfos/nightingale/v6/pkg/ctx"
	"github.com/ccfos/nightingale/v6/pkg/flashduty"
	"github.com/ccfos/nightingale/v6/pkg/httpx"
	"github.com/ccfos/nightingale/v6/pkg/i18nx"
	"github.com/ccfos/nightingale/v6/pkg/logx"
	"github.com/ccfos/nightingale/v6/pkg/version"
	"github.com/ccfos/nightingale/v6/prom"
	"github.com/ccfos/nightingale/v6/pushgw/idents"
	pushgwrt "github.com/ccfos/nightingale/v6/pushgw/router"
	"github.com/ccfos/nightingale/v6/pushgw/writer"
	"github.com/ccfos/nightingale/v6/storage"
	"github.com/ccfos/nightingale/v6/tdengine"

	"github.com/flashcatcloud/ibex/src/cmd/ibex"
)

/**
configDir 配置文件目录
cryptoKey 加密密钥
*/
func Initialize(configDir string, cryptoKey string) (func(), error) {
	// 初始化配置实例(加载配置文件、主要配置参数前置检查)
	config, err := conf.InitConfig(configDir, cryptoKey)
	if err != nil {
		return nil, fmt.Errorf("failed to init config: %v", err)
	}

	cconf.LoadMetricsYaml(configDir, config.Center.MetricsYamlFile) // 加载指标YML
	cconf.LoadOpsYaml(configDir, config.Center.OpsYamlFile)         // Deprecated 该版本没有实际用途

	cconf.MergeOperationConf()

	logxClean, err := logx.Init(config.Log) // 初始化日志配置
	if err != nil {
		return nil, err
	}

	i18nx.Init(configDir) // 初始化国际化组件配置
	cstats.Init()         // 服务状态指标初始化，center server注册Prometheus监控指标(暴露prometheus格式的指标)
	flashduty.Init(config.Center.FlashDuty)

	// Connecting to a Database
	db, err := storage.New(config.DB) // 连接数据库
	if err != nil {
		return nil, err
	}
	ctx := ctx.NewContext(context.Background(), db, true) // 创建一个当前环境的上下文实例
	migrate.Migrate(db)                                   // db迁移
	models.InitRoot(ctx)                                  // 初始化root用户
	// n9e-center从configs库中读取jwt_signing_key，非center服务则向center服务发起http请求，获取配置
	config.HTTP.JWTAuth.SigningKey = models.InitJWTSigningKey(ctx)
	// 初始化RSA配置 (优先db读取、其次指定路径的文件中读取、若均无则生成一对公钥、私钥，并写入db)
	err = rsa.InitRSAConfig(ctx, &config.HTTP.RSA)
	if err != nil {
		return nil, err
	}

	// v7功能-集成中心初始化
	// 遍历integration下的每个目录，读取配置文件，创建对应的models.BuiltinComponent实例，将元数据写入db(if not exists in db)
	// 以及dashboards 仪表盘、metrics 指标、alerts告警规则 写入db(if not exists in db)
	integration.Init(ctx, config.Center.BuiltinIntegrationsDir)

	// 建立redis连接(支持standalone cluster sentinel三种模式)
	var redis storage.Redis
	redis, err = storage.NewRedis(config.Redis)
	if err != nil {
		return nil, err
	}

	metas := metas.New(redis)        //创建一个metas实例并对其初始化, 主要作用是管理和定时持久化主机元数据（HostMeta）到 Redis 数据库中
	idents := idents.New(ctx, redis) //创建一个idents实例并对其初始化, 主要作用是管理和定时持久化主机元数据的更新时间, 同时将该监控对象的更新时间同步到db中

	syncStats := memsto.NewSyncStats()  // 用于创建并注册两个Prometheus 指标（metrics）
	alertStats := astats.NewSyncStats() // 用于创建并注册告警相关的指标

	// Sync Rules From DB To LocalCache
	// 1. user_variables 用户变量
	configCache := memsto.NewConfigCache(ctx, syncStats, config.HTTP.RSA.RSAPrivateKey, config.HTTP.RSA.RSAPassWord)
	// 2. busi_groups 业务组
	busiGroupCache := memsto.NewBusiGroupCache(ctx, syncStats)
	// 3. targets 监控目标对象
	targetCache := memsto.NewTargetCache(ctx, syncStats, redis)
	// 4. datasources 数据源
	dsCache := memsto.NewDatasourceCache(ctx, syncStats)
	// 5. alert_mutes 屏蔽规则
	alertMuteCache := memsto.NewAlertMuteCache(ctx, syncStats)
	//  6. alert_rules 告警规则
	alertRuleCache := memsto.NewAlertRuleCache(ctx, syncStats)
	// 7. 周期性读取 webhooks、smtp、notify_script的配置
	notifyConfigCache := memsto.NewNotifyConfigCache(ctx, configCache)
	// 8. users 用户数据
	userCache := memsto.NewUserCache(ctx, syncStats)
	// 9. user_groups
	userGroupCache := memsto.NewUserGroupCache(ctx, syncStats)
	// 10. 告警自愈任务及脚本
	taskTplCache := memsto.NewTaskTplCache(ctx)

	sso := sso.Init(config.Center, ctx, configCache)
	promClients := prom.NewPromClient(ctx)
	tdengineClients := tdengine.NewTdengineClient(ctx, config.Alert.Heartbeat)

	externalProcessors := process.NewExternalProcessors()
	alert.Start(config.Alert, config.Pushgw, syncStats, alertStats, externalProcessors, targetCache, busiGroupCache, alertMuteCache, alertRuleCache, notifyConfigCache, taskTplCache, dsCache, ctx, promClients, tdengineClients, userCache, userGroupCache)

	writers := writer.NewWriters(config.Pushgw)

	go version.GetGithubVersion()

	alertrtRouter := alertrt.New(config.HTTP, config.Alert, alertMuteCache, targetCache, busiGroupCache, alertStats, ctx, externalProcessors)
	centerRouter := centerrt.New(config.HTTP, config.Center, config.Alert, cconf.Operations, dsCache, notifyConfigCache, promClients, tdengineClients,
		redis, sso, ctx, metas, idents, targetCache, userCache, userGroupCache)
	pushgwRouter := pushgwrt.New(config.HTTP, config.Pushgw, config.Alert, targetCache, busiGroupCache, idents, metas, writers, ctx)

	r := httpx.GinEngine(config.Global.RunMode, config.HTTP)

	centerRouter.Config(r)
	alertrtRouter.Config(r)
	pushgwRouter.Config(r)
	dumper.ConfigRouter(r)

	if config.Ibex.Enable {
		migrate.MigrateIbexTables(db)
		ibex.ServerStart(true, db, redis, config.HTTP.APIForService.BasicAuth, config.Alert.Heartbeat, &config.CenterApi, r, centerRouter, config.Ibex, config.HTTP.Port)
	}

	httpClean := httpx.Init(config.HTTP, r)

	return func() {
		logxClean()
		httpClean()
	}, nil
}
