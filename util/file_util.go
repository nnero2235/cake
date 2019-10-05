package util

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

const (
	FileRWRAll = 777
	FileRWOwner = 644
	FileRWROwner = 744
	FileRAll = 444
)

const (
	SizeGB = 1024 * 1024 * 1024
	SizeMB = 1024 * 1024
	SizeKB = 1024
)

const WindowsDefaultDisk string = "F:"

//return format readable size string
func GetFormatFileSize(nBytes int64) string {
	gb := nBytes / SizeGB
	if gb > 0 {
		gbDouble := float64(nBytes) / SizeGB
		return fmt.Sprintf("%.2fGB",gbDouble)
	}
	mb := nBytes /SizeMB
	if mb > 0 {
		mbDouble := float64(nBytes) / SizeMB
		return fmt.Sprintf("%.2fMB",mbDouble)
	}
	kb := nBytes /SizeKB
	if kb > 0 {
		kbDouble := float64(nBytes) / SizeKB
		return fmt.Sprintf("%.2fKB",kbDouble)
	}
	return fmt.Sprintf("%dB",nBytes)
}

//get different os path in different os platform
func GetOSFilePath(dirs... string) string {
	var rDirs []string
	if runtime.GOOS == "windows" {
		rDirs = []string{WindowsDefaultDisk}
	} else {
		rDirs = []string{"/"}
	}
	return strings.Join(append(rDirs,dirs...),string(os.PathSeparator))
}