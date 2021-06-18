// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"go.uber.org/zap/zapcore"
)

// An udsCore write entries to an UDS server
type udsCore struct {
	client       http.Client
	minimumLevel zapcore.Level
	url          string
	maxAttempts  int
}

// teeToUDSServer returns a zapcore.Core that writes entries to both the provided core and to an uds server.
func teeToUDSServer(baseCore zapcore.Core, address, path string, maxRetryAttempts int) zapcore.Core {
	c := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", address)
			},
		},
		Timeout: 100 * time.Millisecond,
	}
	uc := &udsCore{client: c, url: "http://unix" + path, maxAttempts: maxRetryAttempts + 1}
	for l := zapcore.DebugLevel; l <= zapcore.FatalLevel; l++ {
		if baseCore.Enabled(l) {
			uc.minimumLevel = l
			break
		}
	}
	return zapcore.NewTee(baseCore, uc)
}

// Enabled implements zapcore.Core.
func (u *udsCore) Enabled(l zapcore.Level) bool {
	return l >= u.minimumLevel
}

// With implements zapcore.Core.
func (u *udsCore) With(fields []zapcore.Field) zapcore.Core {
	return &udsCore{
		client:       u.client,
		minimumLevel: u.minimumLevel,
	}
}

// Check implements zapcore.Core.
func (u *udsCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if u.Enabled(e.Level) {
		return ce.AddCore(e, u)
	}
	return ce
}

// Sync implements zapcore.Core.
func (u *udsCore) Sync() error {
	return nil
}

// Write implements zapcore.Core. It writes a log entry to an UDS server.
func (u *udsCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	attempts := 0
	b := backoff.NewExponentialBackOff()
	var lastError error
	for attempts < u.maxAttempts {
		attempts++
		resp, err := u.client.Post(u.url, "text/plain", strings.NewReader(entry.Message))
		if err != nil {
			// Reties on intermittent errors, in case of server restarts etc.
			lastError = fmt.Errorf("error writing logs to uds server %v: %v", u.url, err)
			if attempts < u.maxAttempts {
				time.Sleep(b.NextBackOff())
			}
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("uds server returns non-ok status %v: %v", u.url, resp.Status)
		}
		lastError = nil
		break
	}
	if lastError != nil {
		return lastError
	}
	return nil
}
