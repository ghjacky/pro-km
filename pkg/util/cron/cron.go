/*
Copyright 2020 The Maya Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cron

import (
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// NextTimeFormat cron next time format
const NextTimeFormat = "2006/01/02 15:04:05"

// Cron the cron scheduler struct
type Cron struct {
	scheduler *cron.Cron
	lock      sync.Mutex
	ids       map[uint64]cron.EntryID
}

// NewCron build a new cron scheduler
func NewCron() *Cron {
	return &Cron{
		scheduler: cron.New(),
		ids:       make(map[uint64]cron.EntryID),
	}
}

// Start start the cron scheduler
func (c *Cron) Start() {
	c.scheduler.Start()
}

// Stop stop the cron scheduler
func (c *Cron) Stop() {
	c.scheduler.Stop()
}

// AddCron add a cron job to scheduler
func (c *Cron) AddCron(id uint64, cron string, f func()) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if eid, ok := c.ids[id]; ok {
		c.scheduler.Remove(eid)
	}

	eid, err := c.scheduler.AddFunc(cron, f)
	if err != nil {
		return err
	}
	c.ids[id] = eid
	return nil
}

// RemoveCron remove cron job
func (c *Cron) RemoveCron(id uint64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	eid, ok := c.ids[id]
	if !ok {
		return
	}
	c.scheduler.Remove(eid)
	delete(c.ids, id)
}

// ParseCron parse a string cron spec
func ParseCron(standardSpec string) (cron.Schedule, error) {
	return cron.ParseStandard(standardSpec)
}

// NextTime return the next schedule time
func NextTime(standardSpec string) (time.Time, error) {
	sch, err := cron.ParseStandard(standardSpec)
	if err != nil {
		return time.Time{}, err
	}
	return sch.Next(time.Now()), nil
}

// NextTimeStr return time formatted string
func NextTimeStr(standardSpec string) (string, error) {
	t, err := NextTime(standardSpec)
	if err != nil {
		return "", err
	}
	return t.Format(NextTimeFormat), nil
}
