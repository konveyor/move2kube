/*
 *  Copyright IBM Corporation 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package common

import (
	"context"

	"github.com/sirupsen/logrus"
)

// CleanupHook calls the cleanup functions on fatal and panic errors
type CleanupHook struct {
	ctxContextFn func()
}

// NewCleanupHook creates a cleanup hook
func NewCleanupHook(ctxContextFn context.CancelFunc) *CleanupHook {
	return &CleanupHook{ctxContextFn}
}

// Fire calls the clean up
func (hook *CleanupHook) Fire(entry *logrus.Entry) error {
	hook.ctxContextFn()
	return nil
}

// Levels returns the levels on which the cleanup hook gets called
func (hook *CleanupHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
	}
}
