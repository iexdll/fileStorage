package params

var folderUpload string
var folderStorage string

func SetFolderUpload(value string) {
	folderUpload = value
}

func GetFolderUpload() string {
	return folderUpload
}

func SetFolderStorage(value string) {
	folderStorage = value
}

func GetFolderStorage() string {
	return folderStorage
}
