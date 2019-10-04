package util

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

//return format readable size string
//func GetFileSize(nBytes int) string {
//	gb := nBytes / SizeGB
//	if gb > 0 {
//		mb := nBytes % SizeGB / SizeMB
//	}
//}