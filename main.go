package main

import (
	"context"
	"time"
)

func main() {
	for {
		err := watchServices(context.Background(), dockerClient)
		if err != nil {
			log.Error("watch services docker", err)
			continue
		}
		time.Sleep(time.Duration(ScheduleAtInSecond) * time.Second)
	}
}
