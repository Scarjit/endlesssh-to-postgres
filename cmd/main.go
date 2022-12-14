package main

import (
	"github.com/united-manufacturing-hub/umh-utils/logger"
	"go.uber.org/zap"
	"parseEndlessSSH/cmd/database"
	"parseEndlessSSH/cmd/logparser"
	"time"
)

var buildtime string

func main() {
	initLogging()
	zap.S().Infof("Build time: %s", buildtime)

	db := database.OpenDatabase()
	database.InitDatabase()
	defer database.CloseDatabase()

	for {
		zap.S().Info("Starting logparser")
		err := logparser.ParseLog(db)
		if err != nil {
			zap.S().Fatal(err)
		}
		time.Sleep(1 * time.Hour)
	}
}

func initLogging() {
	_ = logger.New("LOGGING_LEVEL")
}
