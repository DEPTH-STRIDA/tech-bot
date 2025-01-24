package easycodeapi

import (
	"easycodeapp/internal/config"
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/pkg/easycodeapi"
)

var Api *easycodeapi.ApiClient

func init() {
	var err error
	Api, err = easycodeapi.NewEasyCodeApi(easycodeapi.Config{
		AccessToken:     config.File.EasyCodeApiConfig.AccessToken,
		MemberAPIURL:    config.File.EasyCodeApiConfig.MemberAPIURL,
		LessonsAPIURL:   config.File.EasyCodeApiConfig.LessonsAPIURL,
		ApiRequestPause: config.File.EasyCodeApiConfig.ApiRequestPause,
		ApiBufferSize:   config.File.EasyCodeApiConfig.ApiBufferSize,
		Logger:          logger.Log,
	})
	if err != nil {
		panic(err)
	}
}
