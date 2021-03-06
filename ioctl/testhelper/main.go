package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ghishadow/color"
	"golang.org/x/sys/unix"
)

func MountSys(fsType, path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		fmt.Printf("Failed to create directory to mount %v: %v\n", fsType, err)
		unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
	}
	if err := unix.Mount(fsType, path, fsType, unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, ""); err != nil {
		fmt.Printf("Failed to mount %v: %v\n", fsType, err)
		unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
	}
}

func main() {
	if os.Getpid() == 1 { // Running as Init
		os.MkdirAll("/dev", 0755)
		err := unix.Mount("none", "/dev", "devtmpfs", unix.MS_NOSUID, "")
		if err != nil {
			fmt.Printf("Failed to mount /dev: %v\n", err)
			return
		}
		MountSys("tmpfs", "/dev/shm")
		MountSys("sysfs", "/sys")
		MountSys("proc", "/proc")
		color.NoColor = false
		color.Green("Starting tests")
		cmd := exec.Command("/ioctl.test", "-test.v")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err == nil {
			f, err := os.Create("/successful")
			if err != nil {
				fmt.Printf("Failed to write test status: %v", err)
				return
			}
			f.Close()
		}
		color.Blue("Tests completed, grabbing filtered ZFS debug messages")
		debugMessages, err := os.Open("/proc/spl/kstat/zfs/dbgmsg")
		if err != nil {
			color.Red("Failed to open debug messages: %v", err)
			return
		}
		out, err := os.Create("zfsdebug.log")
		if err != nil {
			color.Red("Failed to open debug artifact")
			return
		}
		defer out.Close()
		defer debugMessages.Close()
		scanner := bufio.NewScanner(debugMessages)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintln(out, line)
			if strings.Contains(line, "zfs_ioctl.c") {
				fmt.Println(line)
			}
		}
		out.Sync()
		out.Close()
	}
}
