/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package sectorstorage

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"

	logging "github.com/ipfs/go-log/v2"
)

// Logger Instance
var logLayout = logging.Logger("layout")

type layoutConfig struct {
	CheckLayout               bool `json:"checkLayout"`
	LoadConfigFileAfterMinute int  `json:"loadConfigFileAfterMinute"`
	Groups                    []struct {
		ServerName string         `json:"serverName"`
		Workers    []layoutWorker `json:"workers"`
		Sectors    []layoutSector `json:"sectors"`
	} `json:"groups"`
}

type layoutWorker struct {
	WorkerID string `json:"workerId"`
}

type layoutSector struct {
	SectorID              string `json:"sectorId"`
	AllowLocalFullControl bool   `json:"allowLocalFullControl"`
	RemoteServers         []struct {
		ServerName   string `json:"serverName"`
		AllowedTasks string `json:"allowedTasks"`
	} `json:"remoteServers"`
}

// LayoutConfig instance
var cfg *layoutConfig
var lastConfigUpdate time.Time

// readLayoutConfig Read layout config from json file
func readLayoutConfig() (*layoutConfig, error) {
	// error interface
	var err error

	selectTime := time.Now()

	if cfg != nil {
		selectTime = selectTime.
			Add(time.Hour*0 + time.Minute*time.Duration(cfg.LoadConfigFileAfterMinute*-1) + time.Second*0)
	}

	if cfg != nil && lastConfigUpdate.After(selectTime) {
		return cfg, nil
	}

	// load Json file from environment
	configFile, err := os.Open(os.Getenv("LOTUS_MINER_LAYOUT"))
	if err != nil {
		return nil, err
	}

	defer func(configFile *os.File) {
		err := configFile.Close()
		if err != nil {
		}
	}(configFile)

	// convert config data structure to byte
	configToByte, err := ioutil.ReadAll(configFile)
	if err != nil {
		return nil, err
	}

	// unmarshal data and point to cfg
	err = json.Unmarshal(configToByte, &cfg)
	if err != nil {
		return nil, err
	}

	lastConfigUpdate = time.Now()
	logLayout.Info("New layout config loaded")
	return cfg, nil
}

func WorkerHasLayoutAccess(task *workerRequest, wnd *schedWindowRequest) bool {
	selectedTaskSectorID := task.sector.ID.Number.String()
	selectedWorkerID := wnd.worker.String()

	// Load Config layout
	conf, err := readLayoutConfig()
	if err != nil {
		logLayout.Warnf("Layout config load error %v", err.Error())
		return true
	}
	logLayout.Infof("Layout config status = %v", conf.CheckLayout)

	// Check config CheckLayout is enable
	if !conf.CheckLayout {
		return true
	}

	// sgi = sector group index
	// wgi = work group index
	// sConfig = Sector Config
	_, wgi := getWorkerGroup(conf, selectedWorkerID)
	sConfig, sgi := getSectorGroup(conf, selectedTaskSectorID)

	// Sector config not defined
	if sgi == -1 {
		logLayout.Warnf("Config for sector %v not defined", selectedTaskSectorID)
		return false
	}

	// Worker and Sector are on the same group
	if wgi == sgi && sConfig.AllowLocalFullControl {
		logLayout.Infof("Worker %v and Sector %v are on the same group assiging task local", selectedWorkerID, selectedTaskSectorID)
		return true
	}

	if len(sConfig.RemoteServers) != 0 && wgi != -1 {
		for _, rServer := range sConfig.RemoteServers {
			if rServer.ServerName == conf.Groups[wgi].ServerName {
				logLayout.Infof("Worker %v in server %v has access to Sector %v", selectedWorkerID, rServer.ServerName, selectedTaskSectorID)
				logLayout.Infof("Scheduler is required to do the task type %v", task.taskType.Short())

				if rServer.AllowedTasks == "*" {
					logLayout.Infof("Worker %v in server %v has access to Sector %v all task types", selectedWorkerID, rServer.ServerName, selectedTaskSectorID)
					return true
				}

				for _, sTask := range strings.Split(rServer.AllowedTasks, ",") {
					logLayout.Infof("Allowed task type %v for worker %v", sTask, selectedWorkerID)
					if strings.ToLower(sTask) == strings.ToLower(task.taskType.Short()) {
						logLayout.Infof("Worker %v in server %v has access to Sector %v task type %v", selectedWorkerID, rServer.ServerName, selectedTaskSectorID, sTask)
						return true
					}
				}
			}
		}
	}

	return false
}

// getWorkerGroup get workers in same groups
func getWorkerGroup(conf *layoutConfig, workerID string) (layoutWorker, int) {
	for index, group := range conf.Groups {
		for _, worker := range group.Workers {
			if worker.WorkerID == workerID {
				return worker, index
			}
		}
	}
	return layoutWorker{}, -1
}

// getSectorGroup get sectors in same groups
func getSectorGroup(conf *layoutConfig, sectorID string) (layoutSector, int) {
	for index, group := range conf.Groups {
		for _, sector := range group.Sectors {
			if sector.SectorID == sectorID {
				return sector, index
			}
		}
	}
	return layoutSector{}, -1
}
