package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"strings"
)

type DBInfo struct {
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBDBName   string
	DBSslMode  string
	DBTimezone string
}

type GithubInfo struct {
	ApiUrl  string
	Version string
	AuthKey string
}

type RedisInfo struct {
	RedisHost     string
	RedisUser     string
	RedisPassword string
}

type OpenObserve struct {
	Protocol     string
	Entrypoint   string
	IndexName    string
	Organization string
	UserName     string
	Token        string
}

type Config struct {
	GithubInfo
	DBInfo
	RedisInfo
	proxy    bool
	proxyUrl string
	OpenObserve
}

var Conf *Config

func initConfig(configPath string) error {

	if configPath != "" {
		viper.SetConfigFile(configPath)
		viper.SetConfigType("yaml")
	} else {
		_, e := os.Stat("./config.yml")
		if os.IsExist(e) {
			log.WithFields(log.Fields{
				configPath: "./config.yml",
			}).Info("load config from config file")
			viper.AddConfigPath(".")
			viper.SetConfigName("config.yml")
			viper.SetConfigType("yaml")
		} else {
			log.Info("load config from environment settings")
			viper.AutomaticEnv()
			dbHost := viper.GetString("DB_HOST")
			dbPort := viper.GetInt("DB_PORT")
			dbUser := viper.GetString("DB_USER")
			dbPassword := viper.GetString("DB_PASSWORD")
			dbName := viper.GetString("DB_NAME")
			dbSslMode := viper.GetString("DB_SSL_MODE")
			dbTz := viper.GetString("DB_TZ")
			apiUrl := viper.GetString("API_URL")
			apiVersion := viper.GetString("API_VERSION")
			apiAuthKey := viper.GetString("API_AUTH_KEY")
			redisHost := viper.GetString("REDIS_HOST")
			redisUser := viper.GetString("REDIS_USER")
			redisPasswd := viper.GetString("REDIS_PASSWD")
			Conf = &Config{
				GithubInfo: GithubInfo{
					ApiUrl:  apiUrl,
					Version: apiVersion,
					AuthKey: apiAuthKey,
				},
				DBInfo: DBInfo{
					DBHost:     dbHost,
					DBPort:     dbPort,
					DBUser:     dbUser,
					DBPassword: dbPassword,
					DBDBName:   dbName,
					DBSslMode:  dbSslMode,
					DBTimezone: dbTz,
				},
				RedisInfo: RedisInfo{
					RedisHost:     redisHost,
					RedisUser:     redisUser,
					RedisPassword: redisPasswd,
				},
				OpenObserve: OpenObserve{
					Protocol:     viper.GetString("PROTOCOL"),
					Entrypoint:   viper.GetString("ENTRYPOINT"),
					IndexName:    viper.GetString("INDEX_NAME"),
					Organization: viper.GetString("ORGANIZATION"),
					UserName:     viper.GetString("USERNAME"),
					Token:        viper.GetString("TOKEN"),
				},
			}
			return nil
		}
	}
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
		"X-GitHub-Api-Version": c.Version,
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
