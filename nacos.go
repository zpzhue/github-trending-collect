package main

import (
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/spf13/viper"

	"fmt"
	"strings"
)

type BaseConfig struct {
	id       string
	host     string
	port     uint64
	timeout  uint64
	username string
	password string
	level    string
	dataId   string
	groupId  string
}

func getBaseConfig() *BaseConfig {
	parse := viper.New()
	parse.SetEnvPrefix("NACOS")
	parse.AutomaticEnv()

	host := parse.GetString("HOST")
	port := parse.GetUint64("PORT")
	spaceId := parse.GetString("ID")
	timeout := parse.GetUint64("TIMEOUT")
	if timeout == 0 {
		timeout = 15000
	}
	username := parse.GetString("USERNAME")
	password := parse.GetString("PASSWORD")
	level := parse.GetString("LEVEL")
	dataId := parse.GetString("DATA_ID")
	groupId := parse.GetString("GROUP_ID")

	return &BaseConfig{
		id:       spaceId,
		host:     host,
		port:     port,
		timeout:  timeout,
		username: username,
		password: password,
		level:    level,
		dataId:   dataId,
		groupId:  groupId,
	}
}

func InitConfigClient() *viper.Viper {
	baseConfig := getBaseConfig()

	// 创建clientConfig
	clientConfig := constant.ClientConfig{
		NamespaceId:          baseConfig.id, // 如果需要支持多namespace，我们可以创建多个client,它们有不同的NamespaceId。当namespace是public时，此处填空字符串。
		TimeoutMs:            baseConfig.timeout,
		NotLoadCacheAtStart:  true,
		UpdateCacheWhenEmpty: false,
		CacheDir:             "./cache",
		LogDir:               "./log",
		LogLevel:             baseConfig.level,
		Username:             baseConfig.username,
		Password:             baseConfig.password,
	}

	// 至少一个ServerConfig
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr:      baseConfig.host,
			ContextPath: "/nacos",
			Port:        baseConfig.port,
			Scheme:      "http",
		},
	}

	// 创建动态配置客户端的另一种方式 (推荐)
	configClient, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		fmt.Printf("failed to create nacos config client: %s", err.Error())
		panic(err)
	}

	// 从nacos获取对应配置
	config, err := configClient.GetConfig(
		vo.ConfigParam{
			DataId: baseConfig.dataId,
			Group:  baseConfig.groupId,
		},
	)
	if err != nil {
		fmt.Printf("failed get config content from nacos config client: %s", err.Error())
		panic(err)
	}
	configClient.CloseClient()

	parse := viper.New()
	parse.SetConfigType("yaml")
	err = parse.ReadConfig(strings.NewReader(config))
	if err != nil {
		fmt.Printf("failed to parse config with viper: %s", err.Error())
		panic(err)
	}
	return parse
}
