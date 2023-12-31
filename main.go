package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const TrendingUrl = "https://github.com/trending"
const RedisCachePrefix = "trending"

var languageList = []string{"all", "c", "c++", "go", "java", "jupyter-notebook", "python", "javascript", "typescript", "rust", "vue"}

/*
*
获取对应的日期
*/
func getDate(since string) (date string) {
	cstZone := time.FixedZone("GMT", 8*3600)
	t := time.Now().In(cstZone)

	var duration time.Duration
	switch since {
	case "daily":
		duration = 0
	case "weekly":
		duration = time.Duration(t.Weekday() - 1)
	case "monthly":
		duration = time.Duration(t.Day() - 1)
	default:
		panic("unknown since type: " + since)

	}
	return t.Add(-time.Hour * 24 * duration).Format("2006-01-02")
}

func getTrendingList(client *http.Client, sinceType, language string) (repoList [][2]string) {
	var reqUrl string
	if language == "" {
		panic("language type can't be empty!")
	} else if strings.ToLower(language) == "all" {
		reqUrl = TrendingUrl
	} else {
		reqUrl = TrendingUrl + "/" + language
	}

	log.Info("开始请求" + reqUrl)
	request, err := http.NewRequest("GET", reqUrl, nil)
	query := request.URL.Query()
	query.Add("sinceType", sinceType)
	request.URL.RawQuery = query.Encode()

	if err != nil {
		panic(err)
	}
	r, err := client.Do(request)
	log.Debug("请求完成" + reqUrl)

	if err != nil {
		panic(err)
	}
	bodyReader := r.Body

	if r.StatusCode != 200 {
		log.WithFields(log.Fields{"url": reqUrl}).Fatalf("请求GitHub treading列表状态码异常")
	}
	log.Info("请求Github成功")

	body, err := io.ReadAll(bodyReader)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Fatalf("读取Github返回结果失败")
	}
	doc, err := htmlquery.Parse(strings.NewReader(string(body)))
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Fatalf("解析Github Treading网也内容失败")
	}
	re := regexp.MustCompile(`\d+`)
	articles := htmlquery.Find(doc, "//article")
	for _, article := range articles {
		repoEle := htmlquery.FindOne(article, "h2/a/@href")
		startEle := htmlquery.FindOne(article, "div/span[last()]")
		repoStr := htmlquery.SelectAttr(repoEle, "href")
		startStr := htmlquery.InnerText(startEle)
		startStr = strings.TrimSpace(startStr)
		startStr = re.FindString(startStr)

		repoList = append(repoList, [2]string{repoStr, startStr})
	}

	return repoList
}

func getRepositryInfo(client *http.Client, repo string) (repostry Repostry, err error) {
	repoApi := "https://api.github.com/repos" + repo
	log.WithFields(log.Fields{"url": repoApi}).Info("请求仓库")

	// 1. 构建请求
	req, err := http.NewRequest("GET", repoApi, nil)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Fatal("创建Http请求失败")
	}
	headers := Conf.GetGithubAuthHeader()
	for key, value := range headers {
		req.Header.Add(key, value)
	}

	// 2. 开始请求
	response, err := client.Do(req)
	if err != nil {
		log.WithFields(
			log.Fields{
				"repostry": repoApi,
				"error":    err.Error(),
			}).Error("request github api get error")
		return repostry, err
	}

	// 3. 处理结果
	bodyByte, err := io.ReadAll(response.Body)
	if err != nil {
		log.WithFields(
			log.Fields{"repostry": repoApi, "error": err.Error()},
		).Error("read github api result bytes error")
		return repostry, err
	}
	if response.StatusCode != 200 {
		log.WithFields(log.Fields{"response_code": response.StatusCode})
		log.Info("response body: %s" + string(bodyByte))
		log.WithFields(log.Fields{"request_url": repoApi}).Fatal("请求GitHub仓库结果异常")
	}

	// 4. 序列化成结构体
	err = json.Unmarshal(bodyByte, &repostry)
	if err != nil {
		log.WithFields(
			log.Fields{
				"repostry": repoApi,
				"error":    err.Error(),
			},
		).Error("Unmarshal body error")
		return repostry, err
	}
	return repostry, nil
}

func saveTrendingList(client *http.Client, db *gorm.DB, sinceType string) {
	ctx := context.Background()
	rc := getRedisClient()
	var repoMaps = make(map[string][][2]string)

	for _, language := range languageList {
		repoList := getTrendingList(client, sinceType, language)
		if l := len(repoList); l > 0 {
			repoMaps[language] = repoList
		}
	}

	if l := len(repoMaps); l == 0 {
		log.WithFields(log.Fields{"repo_list_length": l})
		syscall.Exit(1)
	}
	var trendingList []Trending
	date := getDate(sinceType)

	for language, repoList := range repoMaps {
		// 获取key对应的所有map数据(即仓库和start信息)
		redisCacheKey := RedisCachePrefix + "_" + sinceType + "_" + language + "_" + date
		cacheRet, err := rc.HGetAll(ctx, redisCacheKey).Result()
		if err == redis.Nil {
			log.WithFields(log.Fields{"key": redisCacheKey}).Debug("the key not has value")
		} else if err != nil {
			log.WithFields(log.Fields{"key": redisCacheKey, "error": err.Error()}).Fatal("get redis value error")
		}

		for _, repo := range repoList {
			// 检查start是否为数字
			star, err := strconv.Atoi(repo[1])
			if err != nil {
				log.WithFields(log.Fields{"start": repo[1], "error": err.Error()}).Error("str convent to int failue")
			}
			oldStrStar, state := cacheRet[repo[0]]
			oldStar, _ := strconv.Atoi(oldStrStar)

			// 比较大小是否存储
			if !state {
				log.WithFields(log.Fields{"repository": repo[0]}).Debug("repostry not exist, will save to cache")
			} else if err != nil {
				log.WithFields(log.Fields{"key": repo[0], "error": err.Error()}).Fatal("get redis value error, skip this repository")
				continue
			} else if oldStar > star {
				log.WithFields(
					log.Fields{
						"old_value":  oldStar,
						"new_value":  star,
						"repository": repo[0],
					},
				).Info("the old value greater than new value, skip this repository")
				continue
			}
			// 添加到redis缓存
			rc.HSet(ctx, redisCacheKey, map[string]interface{}{repo[0]: star})
			trendingList = append(trendingList, Trending{
				Date:     date,
				Repostry: repo[0],
				Stars:    star,
				Since:    sinceType,
				Language: language,
			})
			log.WithFields(log.Fields{"repositry": repo[0]}).Info("save repository to cache successful")
		}
	}

	// 添加到数据库
	db.Save(&trendingList)
	log.Info("save all trending repositry successful!")
}

func saveRepositry2DB(client *http.Client, db *gorm.DB, sinceType string) {
	ctx := context.Background()
	rc := getRedisClient()
	date := getDate(sinceType)

	for _, language := range languageList {
		redisCacheKey := RedisCachePrefix + "_" + sinceType + "_" + language + "_" + date

		ret, err := rc.HGetAll(ctx, redisCacheKey).Result()
		if err != nil {
			log.WithFields(log.Fields{"error": err.Error()}).Fatal("get redis cahce error: ")
		}

		var repositoryList []Repostry
		for key := range ret {
			r, err := getRepositryInfo(client, key)
			if err != nil {
				log.WithFields(log.Fields{"name": key, "error": err.Error()}).Error("获取repository详细信息失败")
				continue
			}
			repositoryList = append(repositoryList, r)
		}
		db.Save(&repositoryList)
		log.WithFields(log.Fields{
			"repositrySize": len(repositoryList),
		}).Info("save repositry list to database successful")
	}

}

func main() {
	taskName := flag.String("task", "trending", "run collect github trending repositry name task or save repository info task or init database(trending/repo/init_db)")
	sinceTypeName := flag.String("since", "daily", "run collect github trending with since params, choice are daily, weekly, monthly")

	flag.Parse()
	task, sinceType := *taskName, *sinceTypeName
	// 1. 初始化配置
	if err := initConfig(""); err != nil {
		log.Fatalf(err.Error())
	}

	// 2. 加载gorm DB
	db := GetDB()

	// 3. 创建http client
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}
	if Conf.proxy {
		// 添加代理地址
		proxyUrl, err := url.Parse(Conf.proxyUrl)
		if err != nil {
			panic(err)
		}
		tr.Proxy = http.ProxyURL(proxyUrl)
	}
	client := &http.Client{Transport: tr}
	if task == "trending" {
		log.WithFields(log.Fields{"sinceType": sinceType}).Info("will run saveTrendingList task .")
		saveTrendingList(client, db, sinceType)
	} else if task == "repo" {
		log.WithFields(log.Fields{"sinceType": sinceType}).Info("will run saveRepositry2DB task .")
		saveRepositry2DB(client, db, sinceType)
	} else if task == "init_db" {
		MigrateDB()
	} else {
		panic("wrong task type " + task + "!")
	}

}
