package helperfuncs

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const TH32CS_SNAPPROCESS = 0x00000002

type WindowsProcess struct {
	ProcessID       int
	ParentProcessID int
	Exe             string
}

func newWindowsProcess(e *syscall.ProcessEntry32) WindowsProcess {
	// Find when the string ends for decoding
	end := 0
	for {
		if e.ExeFile[end] == 0 {
			break
		}
		end++
	}

	return WindowsProcess{
		ProcessID:       int(e.ProcessID),
		ParentProcessID: int(e.ParentProcessID),
		Exe:             syscall.UTF16ToString(e.ExeFile[:end]),
	}
}

func processes() ([]WindowsProcess, error) {
	handle, err := syscall.CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(handle)

	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	// get the first process
	err = syscall.Process32First(handle, &entry)
	if err != nil {
		return nil, err
	}

	results := make([]WindowsProcess, 0, 50)
	for {
		results = append(results, newWindowsProcess(&entry))

		err = syscall.Process32Next(handle, &entry)
		if err != nil {
			// windows sends ERROR_NO_MORE_FILES on last process
			if err == syscall.ERROR_NO_MORE_FILES {
				return results, nil
			}
			return nil, err
		}
	}
}

func KillProcesses(names ...string) error {
	for _, processName := range names {
		processes, err := findProcessesByName(processName)
		if err != nil {
			return fmt.Errorf("Failed to look for process (%s) (%v)", processName, err)
		}

		for _, process := range processes {
			fmt.Println(fmt.Sprintf("Killing process %s [%v]", processName, process.ProcessID))
			kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(process.ProcessID))
			err := kill.Run()
			if err != nil {
				fmt.Println(fmt.Errorf("Error killing process (%v)", err))
			}
		}
	}
	return nil
}

func findProcessesByName(name string) ([]*WindowsProcess, error) {
	processes, err := processes()
	if err != nil {
		return nil, fmt.Errorf("Failed to get processes (%v)", err)
	}

	foundProcesses := make([]*WindowsProcess, 0)
	for _, p := range processes {
		if bytes.Contains([]byte(strings.ToUpper(p.Exe)), []byte(strings.ToUpper(name))) {
			foundProcesses = append(foundProcesses, &p)
		}
	}
	return foundProcesses, nil
}
