package main

import (
	"os"
	"strconv"
)

const (
	ScheduleAt    = 15
	ScheduleAtEnv = "SCHEDULE_AT"
)

var (
	ScheduleAtInSecond = ScheduleAt
)

func init() {
	value := os.Getenv(ScheduleAtEnv)
	if value == "" {
		return
	}

	s, err := strconv.Atoi(value)
	if err != nil {
		log.Error("parse schedule at", err)
		return
	}

	ScheduleAtInSecond = s
}
