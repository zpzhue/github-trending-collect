package main

import (
	"gorm.io/gorm/logger"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Trending struct {
	ID          int32      `json:"id" gorm:"primaryKey;type:bigserial"`
	Date        time.Time  `json:"date" gorm:"type:date"`
	Repository  string     `json:"repository" gorm:"type:varchar(256);foreignKey:full_name"`
	Stars       int        `json:"stars" gorm:"type:integer"`
	Since       string     `json:"since" gorm:"type:varchar(16)"`
	Language    string     `json:"language" gorm:"type:varchar(32)"`
	UpdatedTime *time.Time `json:"update_time" gorm:"default:current_timestamp"`
	DeletedTime *time.Time `json:"delete_time" gorm:"default:null"`
}

type Repository struct {
	ID               int           `json:"id" gorm:"primaryKey;type:bigint"`
	NodeID           string        `json:"node_id" gorm:"type:varchar(64)"`
	Name             string        `json:"name" gorm:"type:varchar(64)"`
	FullName         string        `json:"full_name" gorm:"type:varchar(256);not null;uniqueIndex"`
	Private          bool          `json:"private"`
	Owner            *Owner        `json:"owner" gorm:"foreignKey:id"`
	HtmlURL          string        `json:"html_url"  gorm:"type:varchar(512)"`
	Description      string        `json:"description" gorm:"type:text"`
	Fork             bool          `json:"fork"`
	URL              string        `json:"url" gorm:"type:varchar(512)"`
	ForksURL         string        `json:"forks_url" gorm:"type:varchar(256)"`
	EventsURL        string        `json:"events_url" gorm:"type:varchar(256)"`
	LanguagesURL     string        `json:"languages_url" gorm:"type:varchar(256)"`
	DownloadsURL     string        `json:"downloads_url" gorm:"type:varchar(256)"`
	CreatedAt        *time.Time    `json:"created_at"`
	UpdatedAt        *time.Time    `json:"updated_at"`
	PushedAt         *time.Time    `json:"pushed_at"`
	GitURL           string        `json:"git_url" gorm:"type:varchar(256)"`
	CloneURL         string        `json:"clone_url" gorm:"type:varchar(256)"`
	Homepage         string        `json:"homepage" gorm:"type:varchar(256)"`
	Size             int           `json:"size"`
	StargazersCount  int           `json:"stargazers_count"`
	Language         string        `json:"language"  gorm:"type:varchar(32)"`
	HasIssues        bool          `json:"has_issues"`
	HasProjects      bool          `json:"has_projects"`
	HasDownloads     bool          `json:"has_downloads"`
	HasWiki          bool          `json:"has_wiki"`
	HasPages         bool          `json:"has_pages"`
	ForksCount       int           `json:"forks_count"`
	Archived         bool          `json:"archived"`
	Disabled         bool          `json:"disabled"`
	OpenIssuesCount  int           `json:"open_issues_count"`
	License          *License      `json:"license" gorm:"foreignKey:key"`
	Topics           *[]string     `json:"topics" gorm:"type:VARCHAR(32)[]"`
	Visibility       string        `json:"visibility"  gorm:"type:varchar(32)"`
	Organization     *Organization `json:"organization" gorm:"foreignKey:id"`
	SubscribersCount int           `json:"subscribers_count"`
	UpdatedTime      *time.Time    `json:"update_time" gorm:"default:current_timestamp"`
	DeletedTime      *time.Time    `json:"delete_time" gorm:"default:null"`
}

type Owner struct {
	ID          int        `json:"id" gorm:"primaryKey;type:bigint"`
	Login       string     `json:"login" gorm:"type:varchar(64)"`
	NodeID      string     `json:"node_id" gorm:"type:varchar(64)"`
	AvatarURL   string     `json:"avatar_url" gorm:"type:varchar(512)"`
	URL         string     `json:"url" gorm:"type:varchar(512)"`
	HtmlURL     string     `json:"html_url" gorm:"type:varchar(512)"`
	ReposURL    string     `json:"repos_url" gorm:"type:varchar(512)"`
	Type        string     `json:"type" gorm:"type:varchar(32)"`
	SiteAdmin   bool       `json:"site_admin"`
	UpdatedTime *time.Time `json:"update_time" gorm:"default:current_timestamp"`
	DeletedTime *time.Time `json:"delete_time" gorm:"default:null"`
}

type Organization struct {
	ID          int        `json:"id" gorm:"primaryKey;type:bigint"`
	Login       string     `json:"login" gorm:"type:varchar(64)"`
	NodeID      string     `json:"node_id" gorm:"type:varchar(64)"`
	AvatarURL   string     `json:"avatar_url" gorm:"type:varchar(512)"`
	HtmlURL     string     `json:"html_url" gorm:"type:varchar(512)"`
	Type        string     `json:"type" gorm:"type:varchar(32)"`
	SiteAdmin   bool       `json:"site_admin"`
	UpdatedTime *time.Time `json:"update_time" gorm:"default:current_timestamp"`
	DeletedTime *time.Time `json:"delete_time" gorm:"default:null"`
}

type License struct {
	Key         string     `json:"key" gorm:"primaryKey;type:varchar(16)"`
	Name        string     `json:"name" gorm:"type:varchar(64)"`
	SpdxID      string     `json:"spdx_id" gorm:"type:varchar(64)"`
	URL         string     `json:"url" gorm:"type:varchar(512)"`
	NodeID      string     `json:"node_id" gorm:"type:varchar(64)"`
	UpdatedTime *time.Time `json:"update_time" gorm:"default:current_timestamp"`
	DeletedTime *time.Time `json:"delete_time" gorm:"default:null"`
}

var DB *gorm.DB

func Init() *gorm.DB {
	dns := Conf.GetDNS()
	db, err := gorm.Open(postgres.Open(dns), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic("failed to connect database")
	}
	DB = db
	//MigrateDB()

	return db
}

func GetDB() *gorm.DB {
	if DB == nil {
		Init()
	}
	return DB
}

func MigrateDB() {
	owner := &Owner{}
	organization := &Organization{}
	license := &License{}
	err := DB.AutoMigrate(
		&Repository{
			Owner:        owner,
			Organization: organization,
			Topics:       &[]string{"python"},
			License:      license,
		},
		owner,
		organization,
		license,
		&Trending{},
	)
	if err != nil {
		log.Fatal("AutoMigrate BD failue")
	}
}
