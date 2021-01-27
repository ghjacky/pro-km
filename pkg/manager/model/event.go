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

package model

import "time"

// Event event is the event record of training task
type Event struct {
	// ID unique id of record
	ID uint64 `gorm:"primary_key" json:"id"`
	// ObjectType type of object happened on
	ObjectType string
	// ObjectID id of object happened on
	ObjectID string
	// Message event detail information
	Message string

	CreatedAt time.Time  `json:"created_at,omitempty"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`
}
