// Copyright 2024 The Erigon Authors
// This file is part of Erigon.
//
// Erigon is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Erigon is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Erigon. If not, see <http://www.gnu.org/licenses/>.

//go:build linux

package diskutils

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/erigontech/erigon-lib/log/v3"
)

func getDeviceID(path string) (uint64, error) {
	var stat syscall.Stat_t
	err := syscall.Stat(path, &stat)
	if err != nil {
		return 0, fmt.Errorf("error stating path: %v", err)
	}
	return stat.Dev, nil
}

func MountPointForDirPath(dirPath string) string {
	actualPath := SmlinkForDirPath(dirPath)

	devID, err := getDeviceID(actualPath)
	if err != nil {
		return ""
	}

	// Open /proc/self/mountinfo
	mountsFile, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	defer mountsFile.Close()

	// Read mountinfo to find matching device ID
	scanner := bufio.NewScanner(mountsFile)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// Extract device ID from the mountinfo line
		var deviceID uint64
		fmt.Sscanf(fields[4], "%d", &deviceID)
		if deviceID == devID {
			return fields[4]
		}
	}

	return ""
}

func SmlinkForDirPath(dirPath string) string {
	fileInfo, err := os.Lstat(dirPath)
	if err != nil {
		log.Debug("[diskutils] Error getting file info for dir path:", dirPath, "Error:", err)
		return dirPath
	}

	if fileInfo.Mode()&os.ModeSymlink != 0 {
		targetPath, err := os.Readlink(dirPath)
		if err != nil {
			log.Debug("[diskutils] Error getting target path for symlink:", dirPath, "Error:", err)
			return dirPath
		} else {
			return targetPath
		}
	} else {
		return dirPath
	}
}

func DiskInfo(disk string) (string, error) {
	cmd := exec.Command("lsblk", "-o", "NAME,KNAME,PATH,MAJ:MIN,FSAVAIL,FSUSE%,FSTYPE,MOUNTPOINT,LABEL,UUID,SIZE,TYPE,RO,RM,MODEL,SERIAL,STATE,OWNER,GROUP,MODE,ALIGNMENT,MIN-IO,OPT-IO,PHY-SEC,LOG-SEC,ROTA,SCHED,RQ-SIZE,DISC-ALN,DISC-GRAN,DISC-MAX,DISC-ZERO,WSAME,WWN,RAND,PKNAME,HCTL,TRAN,SUBSYSTEMS,REV,VENDOR")

	// Capture the output
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error executing lsblk command: %v", err)
	}

	// Process the output
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	header := true

	for scanner.Scan() {
		line := scanner.Text()

		// Skip the header line
		if header {
			header = false
			continue
		}

		//create map key value NAME,KNAME,PATH,MAJ:MIN,FSAVAIL,FSUSE%,FSTYPE,MOUNTPOINT,LABEL,UUID,SIZE,TYPE,RO,RM,MODEL,SERIAL,STATE,OWNER,GROUP,MODE,ALIGNMENT,MIN-IO,OPT-IO,PHY-SEC,LOG-SEC,ROTA,SCHED,RQ-SIZE,DISC-ALN,DISC-GRAN,DISC-MAX,DISC-ZERO,WSAME,WWN,RAND,PKNAME,HCTL,TRAN,SUBSYSTEMS,REV,VENDOR

		expectedLength := 40
		//split line to array of strings

		array := strings.Fields(line)
		fmt.Println("array len: ", len(array))
		fmt.Println("array mount point is: ", array[7])
		if len(array) != expectedLength {
			fmt.Println("Error: unexpected number of fields in lsblk output: %d", len(array))
			fmt.Println("line: ", array)
		}
		//fmt.Println("line: ", array)
		if array[7] == disk {
			fmt.Println("Final line: ", array)
			return line, nil
		}

		// Check if the line contains the mount point
		//if strings.Contains(line, disk) {
		//	fmt.Println(line)
		//	return line, nil
		//}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading output: %v", err)
	}

	return "unknown", nil
}
