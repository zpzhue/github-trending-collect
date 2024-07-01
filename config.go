package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
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
	ApiUrl  string     `mapstructure:"url" yaml:"url"`
	Version *time.Time `mapstructure:"version" yaml:"version"`
	AuthKey string     `mapstructure:"auth-key" yaml:"auth-key"`
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

func initConfig() {
	parse := InitConfigClient()
	err := parse.Unmarshal(&Conf)
	if err != nil {
		panic(err)
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
		"X-GitHub-Api-Version": c.Version.Format("2006-01-02"),
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
