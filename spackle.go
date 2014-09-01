package main

import (
	"fmt"
	"strings"
	"regexp"
	"os"
	"os/exec"
	"io/ioutil"
	"github.com/cam72cam/go-lumberjack/log"
	"github.com/serenitylinux/libspack/misc"
)


func AskYesNo(question string, def bool) bool {
	yn := "[Y/n]"
	if !def {
		yn = "[y/N]"
	}
	fmt.Printf("%s: %s ", question, yn)
	
	var answer string
	fmt.Scanf("%s", &answer)
	
	yesRgx := regexp.MustCompile("(y|Y|yes|Yes)")
	noRgx := regexp.MustCompile("(n|N|no|No)")
	switch {
		case answer == "":
			return def
		case yesRgx.MatchString(answer):
			return true
		case noRgx.MatchString(answer):
			return false
		default:
			fmt.Println("Please enter Y or N")
			return AskYesNo(question, def)
	}
}

func AskQuestion(question string) string {
	log.Info.Println(question)
	var answer string
	fmt.Scanf("%s", &answer)
	
	answer = strings.TrimSpace(answer)
	if len(answer) > 4 {
		return answer
	} else {
		log.Error.Println("Password must be longer than 4 chars")
		return AskQuestion(question)
	}
}

func RequireRoot() {
	if os.Geteuid() != 0 {
		log.Error.Println("Must be root")
		os.Exit(-1)
	}
}

func RequireProg(progname string) {
	_, err := exec.LookPath(progname)
	if err != nil {
		log.Error.Format("Required program not found: %s", err)
		os.Exit(-1)
	}
}

type Device struct {
	file string
	label string
	fstype string
}
func (device *Device) Parent() string{
	return device.file[0:len(device.file)-1]
}

func Blkid() []Device {
	devices := make([]Device, 0)
	
	str, err := misc.RunCommandToString(exec.Command("blkid"))
	if err != nil {
		log.Error.Format("Unable to detect block devices: %s", err)
		os.Exit(-1)
	}
	
	lines := strings.Split(str, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		items := strings.Split(line, " ")
		
		
		var device Device
		device.label = "Unknown"
		for i, item := range items {
			if i == 0 {
				device.file = strings.TrimRight(item, ":")
			} else {
				switch {
					case strings.HasPrefix(item, "LABEL="):
						device.label = strings.Trim(strings.TrimPrefix(item, "LABEL="), "\"")
					case strings.HasPrefix(item, "TYPE="):
						device.fstype = strings.Trim(strings.TrimPrefix(item, "TYPE="), "\"")
				}
			}
		}
		
		
		devices = append(devices, device)
	}
	
	return devices
}

func SelectDevice() Device {

	log.Info.Println("Please select a partition to install to:")
	devices := Blkid()
	
	if len(devices) == 0 {
		log.Error.Println("Unable to find any suitable partitions")
		os.Exit(-1)
	}
	
	for i, device := range devices {
		log.Debug.Format("%d: %s (%s) %s", i+1, device.file, device.label, device.fstype)
	}
	
	var answer int
	fmt.Scanf("%d", &answer)
	if answer > 0 && answer <= len(devices) {
		answer--
		return devices[answer]
	} else {
		log.Error.Println("Invalid Selection");
		return SelectDevice()
	}
}

func FormatDevice(device Device) {
	err := misc.RunCommandToStdOutErr(exec.Command("mkfs.ext4", device.file))
	if err != nil {
		log.Error.Println("Unable to create fs")
		os.Exit(-1)
	}
}

func MountDevice(device Device) string {
	dir, _ := ioutil.TempDir(os.TempDir(), "spackle")
	os.MkdirAll(dir, 755)
	err := misc.RunCommandToStdOutErr(exec.Command("mount", device.file, dir))
	if err != nil {
		log.Error.Println("Unable to mount")
		os.Exit(-1)
	}
	return dir + "/"
}

func InstallTo(dir string, grub bool, device string) {
	err := misc.RunCommandToStdOutErr(exec.Command("spack", "wield", "base", "dhcpcd", "iproute2", "--destdir=" + dir))
	if err != nil {
		log.Error.Println("Error installing base packages")
		os.Exit(-1)
	}
	
	if grub {
		err := misc.RunCommandToStdOutErr(exec.Command("spack", "wield", "grub", "--destdir=" + dir))
		if err != nil {
			log.Error.Println("Error installing grub")
			os.Exit(-1)
		}
		
		err = misc.RunCommandToStdOutErr(exec.Command("mkdir", dir + "/proc"))
		if err != nil {
			log.Error.Println("Unable to create proc")
			os.Exit(-1)
		}
		
		err = misc.RunCommandToStdOutErr(exec.Command("mount", "-t", "proc", "none", dir + "/proc"))
		if err != nil {
			log.Error.Println("Unable to mount proc")
			os.Exit(-1)
		}
		
		err = misc.RunCommandToStdOutErr(exec.Command("mount", "--rbind", "/dev" , dir + "/dev"))
		if err != nil {
			log.Error.Println("Unable to mount dev")
			os.Exit(-1)
		}
		
		misc.RunCommandToStdOutErr(exec.Command("sed", "-i", "s#set -e##", dir + "/etc/grub.d/10_serenity"))
		err = misc.RunCommandToStdOutErr(exec.Command("chroot", dir, "grub-install", device))
		if err != nil {
			log.Error.Println("Unable to install grub")
			os.Exit(-1)
		}
		
		err = misc.RunCommandToStdOutErr(exec.Command("chroot", dir, "bash", "-c", "echo here; grub-mkconfig > /boot/grub/grub.cfg"))
		if err != nil {
			log.Error.Println("Unable to setup grub")
			os.Exit(-1)
		}
	}
}

func SetRootPass(dir, pass string) {
	inner := "echo 'root:%s' | chpasswd"
	inner = fmt.Sprintf(inner, pass)
	err := misc.RunCommandToStdOutErr(exec.Command("chroot", dir, "bash", "-c", inner))
	if err != nil {
		log.Error.Println("Cannot set password")
		os.Exit(-1)
	}
}

func main() {
	log.SetLevel(log.DebugLevel)

	RequireRoot()
	
//	RequireProg("chpasswd")
	RequireProg("chroot")
	RequireProg("blkid")
	RequireProg("mount")
	RequireProg("mkfs.ext4")
	
	log.Info.Println("Welcome to the Serenity Linux Installer")
	misc.LogBar(log.Info, log.Info.Color)
	
	device := SelectDevice()
	doFormat := AskYesNo(fmt.Sprintf("Do you wish to format %s with ext4?", device.file), true)
	doGrub := AskYesNo(fmt.Sprintf("Do you wish to install grub on %s?", device.Parent()), true)
	
	rootPass := AskQuestion("Please choose a root password")
	
	ok := AskYesNo("Are you sure you wish to continue?", true)
	if !ok {
		os.Exit(-1)
	}
	
	if doFormat {
		FormatDevice(device)
	}
	
	dir := MountDevice(device)
	
	InstallTo(dir, doGrub, device.Parent())
	
	SetRootPass(dir, rootPass)
}
