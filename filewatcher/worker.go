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

package filewatcher

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type worker struct {
	mu sync.RWMutex

	// watcher is an fsnotify watcher that watches the parent
	// dir of watchedFiles.
	dirWatcher *fsnotify.Watcher

	// The worker maintain a map of channels keyed by watched file path.
	// The worker watches parent path of given path,
	// and filters out events of given path, then redirect
	// to the result channel.
	// Note that for symlink files, the content in received events
	// do not have to be related to the file itself.
	watchedFiles map[string]*fileTracker

	// add a new path to watch
	addPathCh chan string

	// remove a path
	removePathCh chan string

	// tells the worker to exit
	terminateCh chan bool

	// synchronize with worker activity
	barrierCh chan bool

	funcs *patchTable
}

type fileTracker struct {
	events chan fsnotify.Event
	errors chan error

	// md5 sum to indicate if a file has been updated.
	md5Sum []byte
}

func newWorker(path string, funcs *patchTable) (*worker, error) {
	dirWatcher, err := funcs.newWatcher()
	if err != nil {
		return nil, err
	}

	if err = funcs.addWatcherPath(dirWatcher, path); err != nil {
		_ = dirWatcher.Close()
		return nil, err
	}

	wk := &worker{
		dirWatcher:   dirWatcher,
		watchedFiles: make(map[string]*fileTracker),
		addPathCh:    make(chan string),
		removePathCh: make(chan string),
		terminateCh:  make(chan bool),
		barrierCh:    make(chan bool),
		funcs:        funcs,
	}

	go wk.listen()

	return wk, nil
}

func (wk *worker) listen() {
	wk.loop()

	_ = wk.dirWatcher.Close()

	// clean up
	for _, ft := range wk.watchedFiles {
		close(ft.errors)
		close(ft.events)
	}
}

func (wk *worker) loop() {
	for {
		select {
		case event := <-wk.dirWatcher.Events:
			for path, ft := range wk.watchedFiles {
				sum := getMd5Sum(path)
				if !bytes.Equal(sum, ft.md5Sum) {
					ft.md5Sum = sum

					select {
					case ft.events <- event:
						// nothing to do

					case path := <-wk.addPathCh:
						wk.registerPath(path)

					case path := <-wk.removePathCh:
						wk.unregisterPath(path)

					case <-wk.terminateCh:
						return

					case <-wk.barrierCh:
						// nothing to do
					}
				}
			}

		case err := <-wk.dirWatcher.Errors:
			for _, ft := range wk.watchedFiles {
				select {
				case ft.errors <- err:

				case path := <-wk.addPathCh:
					wk.registerPath(path)

				case path := <-wk.removePathCh:
					wk.unregisterPath(path)

				case <-wk.terminateCh:
					return

				case <-wk.barrierCh:
					// nothing to do
				}
			}

		case path := <-wk.addPathCh:
			wk.registerPath(path)

		case path := <-wk.removePathCh:
			wk.unregisterPath(path)

		case <-wk.terminateCh:
			return

		case <-wk.barrierCh:
			// nothing to do
		}
	}
}

// called from within the goroutine
func (wk *worker) registerPath(path string) {
	ft := wk.watchedFiles[path]
	if ft != nil {
		wk.funcs.panic(fmt.Sprintf("can't watch the %s path multiple times", path))
		return
	}

	ft = &fileTracker{
		events: make(chan fsnotify.Event),
		errors: make(chan error),
		md5Sum: getMd5Sum(path),
	}

	wk.mu.Lock()
	wk.watchedFiles[path] = ft
	wk.mu.Unlock()
}

// called from within the goroutine
func (wk *worker) unregisterPath(path string) {
	ft := wk.watchedFiles[path]
	if ft == nil {
		wk.funcs.panic(fmt.Sprintf("can't stop watching the %s path as it wasn't being watched", path))
		return
	}

	wk.mu.Lock()
	delete(wk.watchedFiles, path)
	wk.mu.Unlock()
	close(ft.errors)
	close(ft.events)
}

func (wk *worker) terminate() {
	wk.terminateCh <- true
}

func (wk *worker) addPath(path string) {
	wk.addPathCh <- path
}

func (wk *worker) removePath(path string) {
	wk.removePathCh <- path
}

func (wk *worker) eventChannel(path string) chan fsnotify.Event {
	wk.sync()

	wk.mu.RLock()
	defer wk.mu.RUnlock()

	if ft, ok := wk.watchedFiles[path]; ok {
		return ft.events
	}

	return nil
}

func (wk *worker) errorChannel(path string) chan error {
	wk.sync()

	wk.mu.RLock()
	defer wk.mu.RUnlock()

	if ft, ok := wk.watchedFiles[path]; ok {
		return ft.errors
	}

	return nil
}

func (wk *worker) sync() {
	// Ensure any previous add/remove has completed
	//
	// Since we're using blocking channels, the caller blocks when adding or removing a path
	// until the worker goroutine has woken up and started processing the path. Poking this
	// barrier will block until the worker can get around to reading the message from the channel,
	// which indirectly ensures that a previous addPath/removePath has already completed and so the
	// event and error channels have been created
	wk.barrierCh <- true
}

// gets the MD5 of the given file, or nil if there's a problem
func getMd5Sum(file string) []byte {
	f, err := os.Open(file)
	if err != nil {
		return nil
	}
	defer f.Close()
	r := bufio.NewReader(f)

	h := md5.New()
	_, _ = io.Copy(h, r)
	return h.Sum(nil)
}
