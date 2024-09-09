package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type DBInfo struct {
	DBHost     string `mapstructure:"host" yaml:"host"`
	DBPort     int    `mapstructure:"port" yaml:"port"`
	DBUser     string `mapstructure:"user" yaml:"user"`
	DBPassword string `mapstructure:"password" yaml:"password"`
	DBDBName   string `mapstructure:"name" yaml:"name"`
	DBSslMode  string `mapstructure:"ssl-mode" yaml:"ssl-mode"`
	DBTimezone string `mapstructure:"timezone" yaml:"timezone"`
}

type GithubInfo struct {
	ApiUrl  string `mapstructure:"url" yaml:"url"`
	Version string `mapstructure:"version" yaml:"version"`
	AuthKey string `mapstructure:"auth-key" yaml:"auth-key"`
}

type RedisInfo struct {
	RedisHost     string `mapstructure:"host" yaml:"host"`
	RedisUser     string `mapstructure:"user" yaml:"user"`
	RedisPassword string `mapstructure:"password" yaml:"password"`
}

type OpenObserve struct {
	Protocol     string `mapstructure:"protocol" yaml:"protocol"`
	Entrypoint   string `mapstructure:"entrypoint" yaml:"entrypoint"`
	IndexName    string `mapstructure:"index-name" yaml:"index-name"`
	Organization string `mapstructure:"organization" yaml:"organization"`
	UserName     string `mapstructure:"username" yaml:"username"`
	Token        string `mapstructure:"token" yaml:"token"`
}

type Config struct {
	GithubInfo  `mapstructure:"github" yaml:"github"`
	DBInfo      `mapstructure:"db" yaml:"db"`
	RedisInfo   `mapstructure:"redis" yaml:"redis"`
	proxy       bool   `mapstructure:"proxy" yaml:"proxy"`
	proxyUrl    string `mapstructure:"proxy-url" yaml:"proxy-url"`
	OpenObserve `mapstructure:"open-observe" yaml:"open-observe"`
}

var Conf *Config

func initConfig() error {
	_, e := os.Stat("./config.yaml")
	if e == nil {
		log.Info("found local config file ,loading config data ...")
		viper.AddConfigPath(".")
		viper.SetConfigName("config.yaml")
		viper.SetConfigType("yaml")

		viper.AutomaticEnv()
		replacer := strings.NewReplacer(".", "_")
		viper.SetEnvKeyReplacer(replacer)
		if err := viper.ReadInConfig(); err != nil {
			return errors.WithStack(err)
		}

		err := viper.Unmarshal(&Conf)
		if err != nil {
			return err
		}

		watchConfig()
		log.Info("init config finished. ")
		return nil
	} else {
		parse := InitConfigClient()
		err := parse.Unmarshal(&Conf)
		if err != nil {
			return err
		}
		return nil
	}
}

func watchConfig() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.WithFields(log.Fields{"name": e.Name}).Info("Config file changed")
	})
}

func (c *Config) GetDNS() (dns string) {
	dns = fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		c.DBHost, c.DBUser, c.DBPassword, c.DBDBName, c.DBPort, c.DBSslMode, c.DBTimezone,
	)
	return dns
}

func (c *Config) GetGithubAuthHeader() map[string]string {
	return map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": c.GithubInfo.Version,
		"Authorization":        "Bearer " + c.AuthKey,
	}
}

func getRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:                  Conf.RedisHost,
		Username:              Conf.RedisUser,
		Password:              Conf.RedisPassword,
		ContextTimeoutEnabled: false,
	})

	return client
}
