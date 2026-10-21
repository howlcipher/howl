//go:build windows

package interop

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Start suspended, assign to a non-breakaway kill-on-close job, then resume.
// Assigning a running process would race descendants created before assignment.
func configureProcessTree(cmd *exec.Cmd) (func() error, func(), error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, nil, err
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		windows.CloseHandle(job)
		return nil, nil, err
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_SUSPENDED}
	var once sync.Once
	cleanup := func() { once.Do(func() { _ = windows.TerminateJobObject(job, 1); _ = windows.CloseHandle(job) }) }

	cmd.Cancel = func() error {
		_ = windows.TerminateJobObject(job, 1)
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return nil
	}
	started := func() error {
		process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
		if err != nil {
			return err
		}
		defer windows.CloseHandle(process)
		if err = windows.AssignProcessToJobObject(job, process); err != nil {
			return err
		}
		snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
		if err != nil {
			return err
		}
		defer windows.CloseHandle(snapshot)
		entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
		for err = windows.Thread32First(snapshot, &entry); err == nil; err = windows.Thread32Next(snapshot, &entry) {
			if entry.OwnerProcessID != uint32(cmd.Process.Pid) {
				continue
			}
			thread, e := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
			if e != nil {
				return e
			}
			_, e = windows.ResumeThread(thread)
			windows.CloseHandle(thread)
			return e
		}
		return fmt.Errorf("descriptor suspended thread not found: %w", err)
	}
	return started, cleanup, nil
}
