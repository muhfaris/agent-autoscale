package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

var (
	dockerClient      *client.Client
	labelAutoScale    = "swarm.autoscale=true"
	labelAutoScaleMin = "swarm.autoscale.min"
	labelAutoScaleMax = "swarm.autoscale.max"
	labelAutoScaleCPU = "swarm.autoscale.cpu"
	labelAutoScaleMem = "swarm.autoscale.mem"
)

// Stats contains the statistics of a currently running Docker container.
type Stats struct {
	Container string      `json:"container"`
	Name      string      `json:"name"`
	Memory    MemoryStats `json:"memory"`
	CPU       string      `json:"cpu"`
	IO        IOStats     `json:"io"`
	PIDs      int         `json:"pids"`
}

// String returns a human-readable string containing the details of a Stats value.
func (s Stats) String() string {
	return fmt.Sprintf("Container=%v Memory={%v} CPU=%v IO={%v} PIDs=%v", s.Container, s.Memory, s.CPU, s.IO, s.PIDs)
}

// MemoryStats contains the statistics of a running Docker container related to
// memory usage.
type MemoryStats struct {
	Raw     string `json:"raw"`
	Percent string `json:"percent"`
}

// String returns a human-readable string containing the details of a MemoryStats value.
func (m MemoryStats) String() string {
	return fmt.Sprintf("Raw=%v Percent=%v", m.Raw, m.Percent)
}

// IOStats contains the statistics of a running Docker container related to
// IO, including network and block.
type IOStats struct {
	Network string `json:"network"`
	Block   string `json:"block"`
}

// String returns a human-readable string containing the details of a IOStats value.
func (i IOStats) String() string {
	return fmt.Sprintf("Network=%v Block=%v", i.Network, i.Block)
}

// StatsResult is the value recieved when using Monitor to listen for
// Docker statistics.
type StatsResult struct {
	Stats []Stats `json:"stats"`
	Error error   `json:"error"`
}

func watchServices(ctx context.Context, cli *client.Client) error {
	services, err := cli.ServiceList(ctx, types.ServiceListOptions{
		Status:  true,
		Filters: filters.NewArgs(filters.Arg("label", labelAutoScale))})
	if err != nil {
		return err
	}

	for _, service := range services {
		if service.ServiceStatus.RunningTasks == 0 {
			log.Warn(fmt.Sprintf("service %s has no running tasks", service.Spec.Name))
			continue
		}

		mapStats, err := watchStats(ctx)
		if err != nil {
			return err
		}

		stats, ok := mapStats[service.Spec.Name]
		if !ok {
			continue
		}
		log.Info("stats:", "service:", service.Spec.Name, "CPU:", stats.CPU, "Memory:", stats.Memory.Percent)

		cpu := strings.ReplaceAll(stats.CPU, "%", "")
		cpuPercent, err := strconv.ParseFloat(cpu, 64)
		if err != nil {
			log.Error("parse cpu percentage", err)
			continue
		}

		mem := strings.ReplaceAll(stats.Memory.Percent, "%", "")
		memPercent, err := strconv.ParseFloat(mem, 64)
		if err != nil {
			log.Error("parse memory percentage", err)
			continue
		}

		var (
			basedCPUPercent     = service.Spec.Labels[labelAutoScaleCPU]
			basedMemPercent     = service.Spec.Labels[labelAutoScaleMem]
			minContainer        = service.Spec.Labels[labelAutoScaleMin]
			maxContainer        = service.Spec.Labels[labelAutoScaleMax]
			basedCPU            float64
			basedMem            float64
			minContainerService int64
			maxContainerService int64
		)

		if basedCPUPercent != "" {
			basedCPU, err = strconv.ParseFloat(basedCPUPercent, 64)
			if err != nil {
				log.Error("parse based cpu percentage", err)
				continue
			}
		}

		if basedMemPercent != "" {
			basedMem, err = strconv.ParseFloat(basedMemPercent, 64)
			if err != nil {
				log.Error("parse based memory percentage", err)
				continue
			}
		}

		if minContainer != "" {
			minContainerService, err = strconv.ParseInt(minContainer, 10, 64)
			if err != nil {
				log.Error("parse min container service", err)
				continue
			}

		}

		if maxContainer != "" {
			maxContainerService, err = strconv.ParseInt(maxContainer, 10, 64)
			if err != nil {
				log.Error("parse max container service", err)
				continue
			}
		}

		containerStats := Container{
			ServiceID:   service.ID,
			ServiceName: service.Spec.Name,
			Current: CurrentStats{
				CPUPercentage:    cpuPercent,
				MemoryPercentage: memPercent,
				Replicas:         int(*service.Spec.Mode.Replicated.Replicas),
			},
			Based: BasedStats{
				CPUPercentage:    basedCPU,
				MemoryPercentage: basedMem,
				Min:              minContainerService,
				Max:              maxContainerService,
			},
		}

		go func() {
			err := sendStats(containerStats)
			if err != nil {
				log.Error("unable to send stats", err)
			}
		}()
	}

	return nil
}

func watchStats(ctx context.Context) (map[string]Stats, error) {
	var (
		dockerPath = "/usr/bin/docker"
		commands   = []string{"stats", "--no-stream", "--format", `{"container":"{{.Container}}","name":"{{.Name}}","memory":{"raw":"{{.MemUsage}}","percent":"{{.MemPerc}}"},"cpu":"{{.CPUPerc}}","io":{"network":"{{.NetIO}}","block":"{{.BlockIO}}"},"pids":{{.PIDs}}}`}
		stats      = make(map[string]Stats)
	)

	out, err := exec.Command(dockerPath, commands...).Output()
	if err != nil {
		return nil, err
	}

	containers := strings.Split(string(out), "\n")
	for _, con := range containers {
		if len(con) == 0 {
			continue
		}

		var s Stats
		if err := json.Unmarshal([]byte(con), &s); err != nil {
			return nil, err
		}

		name := s.Name
		names := strings.Split(name, ".")
		name = names[0]

		stats[name] = s
	}

	return stats, nil
}

func init() {
	const dockerVersion = "1.44"
	cli, err := client.NewClientWithOpts(client.WithVersion(dockerVersion))
	if err != nil {
		log.Error("initialize docker client", err)
		return
	}

	dockerClient = cli
}
