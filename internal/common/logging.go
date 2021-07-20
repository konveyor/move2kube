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
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	// InfoLevelStr stores the info loggin level in string
	InfoLevelStr = "info"
)

// GetLogLevel converts from string logging level to logrus logging level
func GetLogLevel(loglevel string) logrus.Level {
	switch strings.ToLower(loglevel) {
	case "trace":
		return logrus.TraceLevel
	case "debug", "verbose":
		return logrus.DebugLevel
	case InfoLevelStr:
		return logrus.InfoLevel
	case "warn":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	case "panic":
		return logrus.PanicLevel
	default:
		return logrus.InfoLevel
	}
}

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
