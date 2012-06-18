// Copyright (c) 2012 VMware, Inc.

package psnotify

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
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
				events.forks = append(events.forks, ev.ParentPid)
			case ev := <-watcher.Exec:
				events.execs = append(events.execs, ev.Pid)
			case ev := <-watcher.Exit:
				events.exits = append(events.exits, ev.Pid)
			case err := <-watcher.Error:
				events.errors = append(events.errors, err)
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

func expectEvents(t *testing.T, num int, name string, pids []int) bool {
	if len(pids) != num {
		t.Errorf("Expected %d %s events, got=%v", num, name, pids)
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
	cmd := exec.Command("date")
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}

	// watch fork events for this process
	if err := tw.watcher.Watch(pid, PROC_EVENT_FORK); err != nil {
		t.Error(err)
	}

	// this fork event will be captured,
	// the exec and exit events will not be captured
	cmd = exec.Command("cal")
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}

	tw.close()

	if expectEvents(t, 1, "forks", tw.events.forks) {
		if tw.events.forks[0] != pid {
			t.Errorf("Expected fork pid=%d, got=%v",
				pid, tw.events.forks)
		}
	}

	expectEvents(t, 0, "execs", tw.events.execs)
	expectEvents(t, 0, "exits", tw.events.exits)
}

func TestWatchExit(t *testing.T) {
	if skipTest(t) {
		return
	}

	tw := newTestWatcher(t)

	cmd := exec.Command("perl", "-e", "sleep 100")
	if err := cmd.Start(); err != nil {
		t.Error(err)
	}

	childPid := cmd.Process.Pid

	// watch for exit event of our child process
	if err := tw.watcher.Watch(childPid, PROC_EVENT_EXIT); err != nil {
		t.Error(err)
	}

	// kill our child process, triggers exit event
	syscall.Kill(childPid, syscall.SIGTERM)

	cmd.Wait()

	tw.close()

	expectEvents(t, 0, "forks", tw.events.forks)

	expectEvents(t, 0, "execs", tw.events.execs)

	if expectEvents(t, 1, "exits", tw.events.exits) {
		if tw.events.exits[0] != childPid {
			t.Errorf("Expected exit pid=%d, got=%v",
				childPid, tw.events.exits)
		}
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

	cmd := exec.Command("perl", "-e", "sleep 100")
	if err := cmd.Start(); err != nil {
		t.Error(err)
	}

	childPid := cmd.Process.Pid

	if err := tw.watcher.Watch(childPid, PROC_EVENT_EXIT); err != nil {
		t.Error(err)
	}

	syscall.Kill(childPid, syscall.SIGTERM)

	cmd.Wait()

	tw.close()

	if expectEvents(t, 1, "forks", tw.events.forks) {
		if tw.events.forks[0] != pid {
			t.Errorf("Expected fork pid=%d, got=%v",
				pid, tw.events.forks)
		}
	}

	expectEvents(t, 0, "execs", tw.events.execs)

	if expectEvents(t, 1, "exits", tw.events.exits) {
		if tw.events.exits[0] != childPid {
			t.Errorf("Expected exit pid=%d, got=%v",
				childPid, tw.events.exits)
		}
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
		cmd := exec.Command(name)
		if err := cmd.Run(); err != nil {
			t.Error(err)
		}
		childPids[i] = cmd.Process.Pid
	}

	// remove watch for this process
	tw.watcher.RemoveWatch(pid)

	// run commands again to make sure we don't receive any unwanted events
	for _, name := range commands {
		cmd := exec.Command(name)
		if err := cmd.Run(); err != nil {
			t.Error(err)
		}
	}

	tw.close()

	// run commands again to make sure nothing panics after
	// closing the watcher
	for _, name := range commands {
		cmd := exec.Command(name)
		if err := cmd.Run(); err != nil {
			t.Error(err)
		}
	}

	num := len(commands)
	if expectEvents(t, num, "forks", tw.events.forks) {
		for _, epid := range tw.events.forks {
			if epid != pid {
				t.Errorf("Expected fork pid=%d, got=%v",
					pid, epid)
			}
		}
	}

	if expectEvents(t, num, "execs", tw.events.execs) {
		for i, _ := range childPids {
			if tw.events.execs[i] != childPids[i] {
				t.Errorf("Expected fork pid=%d, got=%v",
					childPids[i], tw.events.execs[i])
			}
		}
	}

	if expectEvents(t, num, "exits", tw.events.exits) {
		for i, _ := range childPids {
			if tw.events.exits[i] != childPids[i] {
				t.Errorf("Expected exit pid=%d, got=%v",
					childPids[i], tw.events.exits[i])
			}
		}
	}
}
