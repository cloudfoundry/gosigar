// +build darwin freebsd linux netbsd openbsd

package psnotify

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"
)

type anyEvent struct {
	exits  []int
	forks  []int
	execs  []int
	errors []error
	done   chan bool
	m      sync.Mutex
}

func (a *anyEvent) append(eventType string, event interface{}) {
	a.m.Lock()
	defer a.m.Unlock()

	switch eventType {
	case "exits":
		a.exits = append(a.exits, event.(int))
	case "forks":
		a.forks = append(a.forks, event.(int))
	case "execs":
		a.execs = append(a.execs, event.(int))
	case "errors":
		a.errors = append(a.errors, event.(error))
	}
}

func (a *anyEvent) getExits() []int {
	a.m.Lock()
	defer a.m.Unlock()

	r := make([]int, len(a.exits))
	for i, v := range a.exits {
		r[i] = v
	}
	return r
}

func (a *anyEvent) getForks() []int {
	a.m.Lock()
	defer a.m.Unlock()

	r := make([]int, len(a.forks))
	for i, v := range a.forks {
		r[i] = v
	}
	return r
}

func (a *anyEvent) getExecs() []int {
	a.m.Lock()
	defer a.m.Unlock()

	r := make([]int, len(a.execs))
	for i, v := range a.execs {
		r[i] = v
	}
	return r
}

func (a *anyEvent) getErrors() []error {
	a.m.Lock()
	defer a.m.Unlock()

	r := make([]error, len(a.errors))
	for i, v := range a.errors {
		r[i] = v
	}
	return r
}

type testWatcher struct {
	t       *testing.T
	watcher *Watcher
	events  *anyEvent
}

// General purpose Watcher wrapper for all tests
func newTestWatcher(t *testing.T) *testWatcher {
	watcher, err := NewWatcher()
	if err != nil {
		t.Fatal(err)
	}

	events := &anyEvent{
		done: make(chan bool, 1),
	}

	tw := &testWatcher{
		t:       t,
		watcher: watcher,
		events:  events,
	}

	go func() {
		for {
			select {
			case <-events.done:
				return
			case ev := <-watcher.Fork:
				events.append("forks", ev.ParentPid)
			case ev := <-watcher.Exec:
				events.append("execs", ev.Pid)
			case ev := <-watcher.Exit:
				events.append("exits", ev.Pid)
			case err := <-watcher.Error:
				events.append("errors", err)
			}
		}
	}()

	return tw
}

func (tw *testWatcher) close() {
	pause := 100 * time.Millisecond
	time.Sleep(pause)

	tw.events.done <- true

	tw.watcher.Close()

	time.Sleep(pause)
}

func skipTest(t *testing.T) bool {
	if runtime.GOOS == "linux" && os.Getuid() != 0 {
		fmt.Println("SKIP: test must be run as root on linux")
		return true
	}
	return false
}

func startSleepCommand(t *testing.T) *exec.Cmd {
	cmd := exec.Command("sh", "-c", "sleep 100")
	if err := cmd.Start(); err != nil {
		t.Error(err)
	}
	return cmd
}

func runCommand(t *testing.T, name string) *exec.Cmd {
	cmd := exec.Command(name)
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}
	return cmd
}

func expectEvents(t *testing.T, num int, name string, pids []int) bool {
	if len(pids) != num {
		t.Errorf("Expected %d %s events, got=%v", num, name, pids)
		return false
	}
	return true
}

func expectEventPid(t *testing.T, name string, expect int, pid int) bool {
	if expect != pid {
		t.Errorf("Expected %s pid=%d, received=%d", name, expect, pid)
		return false
	}
	return true
}

func TestWatchFork(t *testing.T) {
	if skipTest(t) {
		return
	}

	pid := os.Getpid()

	tw := newTestWatcher(t)

	// no watches added yet, so this fork event will no be captured
	runCommand(t, "date")

	// watch fork events for this process
	if err := tw.watcher.Watch(pid, PROC_EVENT_FORK); err != nil {
		t.Error(err)
	}

	// this fork event will be captured,
	// the exec and exit events will not be captured
	runCommand(t, "cal")

	tw.close()

	if expectEvents(t, 1, "forks", tw.events.getForks()) {
		expectEventPid(t, "fork", pid, tw.events.getForks()[0])
	}

	expectEvents(t, 0, "execs", tw.events.getExecs())
	expectEvents(t, 0, "exits", tw.events.getExits())
}

func TestWatchExit(t *testing.T) {
	if skipTest(t) {
		return
	}

	tw := newTestWatcher(t)

	cmd := startSleepCommand(t)

	childPid := cmd.Process.Pid

	// watch for exit event of our child process
	if err := tw.watcher.Watch(childPid, PROC_EVENT_EXIT); err != nil {
		t.Error(err)
	}

	// kill our child process, triggers exit event
	syscall.Kill(childPid, syscall.SIGTERM)

	cmd.Wait()

	tw.close()

	expectEvents(t, 0, "forks", tw.events.getForks())

	expectEvents(t, 0, "execs", tw.events.getExecs())

	if expectEvents(t, 1, "exits", tw.events.getExits()) {
		expectEventPid(t, "exit", childPid, tw.events.getExits()[0])
	}
}

// combined version of TestWatchFork() and TestWatchExit()
func TestWatchForkAndExit(t *testing.T) {
	if skipTest(t) {
		return
	}

	pid := os.Getpid()

	tw := newTestWatcher(t)

	if err := tw.watcher.Watch(pid, PROC_EVENT_FORK); err != nil {
		t.Error(err)
	}

	cmd := startSleepCommand(t)

	childPid := cmd.Process.Pid

	if err := tw.watcher.Watch(childPid, PROC_EVENT_EXIT); err != nil {
		t.Error(err)
	}

	syscall.Kill(childPid, syscall.SIGTERM)

	cmd.Wait()

	tw.close()

	if expectEvents(t, 1, "forks", tw.events.getForks()) {
		expectEventPid(t, "fork", pid, tw.events.getForks()[0])
	}

	expectEvents(t, 0, "execs", tw.events.getExecs())

	if expectEvents(t, 1, "exits", tw.events.getExits()) {
		expectEventPid(t, "exit", childPid, tw.events.getExits()[0])
	}
}

func TestWatchFollowFork(t *testing.T) {
	if skipTest(t) {
		return
	}

	// Darwin is not able to follow forks, as the kqueue fork event
	// does not provide the child pid.
	if runtime.GOOS != "linux" {
		fmt.Println("SKIP: test follow forks is linux only")
		return
	}

	pid := os.Getpid()

	tw := newTestWatcher(t)

	// watch for all process events related to this process
	if err := tw.watcher.Watch(pid, PROC_EVENT_ALL); err != nil {
		t.Error(err)
	}

	commands := []string{"date", "cal"}
	childPids := make([]int, len(commands))

	// triggers fork/exec/exit events for each command
	for i, name := range commands {
		cmd := runCommand(t, name)
		childPids[i] = cmd.Process.Pid
	}

	// remove watch for this process
	tw.watcher.RemoveWatch(pid)

	// run commands again to make sure we don't receive any unwanted events
	for _, name := range commands {
		runCommand(t, name)
	}

	tw.close()

	// run commands again to make sure nothing panics after
	// closing the watcher
	for _, name := range commands {
		runCommand(t, name)
	}

	num := len(commands)
	if expectEvents(t, num, "forks", tw.events.getForks()) {
		for _, epid := range tw.events.getForks() {
			expectEventPid(t, "fork", pid, epid)
		}
	}

	if expectEvents(t, num, "execs", tw.events.getExecs()) {
		for i, epid := range tw.events.getExecs() {
			expectEventPid(t, "exec", childPids[i], epid)
		}
	}

	if expectEvents(t, num, "exits", tw.events.getExits()) {
		for i, epid := range tw.events.getExits() {
			expectEventPid(t, "exit", childPids[i], epid)
		}
	}
}
