package main

import (
	log "github.com/sirupsen/logrus"

	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TrendingRecord struct {
	Date       string `json:"date"`
	Repository string `json:"repository"`
	Stars      int    `json:"stars"`
	Since      string `json:"since"`
	Language   string `json:"language"`
	Action     string `json:"action"`
	RepoId     int32  `json:"repoId"`
	Time       string `json:"_time"`
	TimeStamp  int64  `json:"_timestamp"`
}

func fixRecordTime(data []*TrendingRecord) {
	now := time.Now()
	for _, record := range data {
		if record.Time == "" {
			t := now.In(time.FixedZone("GMT", 8*3600))
			record.Time = t.Format(time.RFC3339)
		}
		if record.TimeStamp <= 0 {
			t := now.In(time.UTC)
			record.TimeStamp = t.UnixMicro()
		}
	}
}

// EmitMessage 发送记录的Trending到OpenObserve
func EmitMessage(data []*TrendingRecord) {
	fixRecordTime(data)
	dataBytes, err := json.Marshal(data)

	url := fmt.Sprintf(
		"%s://%s/api/%s/%s/_json",
		Conf.OpenObserve.Protocol,
		Conf.OpenObserve.Entrypoint,
		Conf.OpenObserve.Organization,
		Conf.OpenObserve.IndexName,
	)
	req, err := http.NewRequest("POST", url, bytes.NewReader(dataBytes))
	if err != nil {
		log.WithFields(log.Fields{"message": "build openobserve api error"}).Error(err)
	}
	req.SetBasicAuth(Conf.OpenObserve.UserName, Conf.OpenObserve.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("read openobserve response error")
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(
			log.Fields{
				"message":     string(body),
				"status_code": resp.StatusCode,
			},
		).Error("openobserve api get error result")
	} else {
		log.Info("request openobserve api success")
	}
}
