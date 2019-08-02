package main

import (
	"crypto/rand"
	"encoding/json"
	"fileStorage/mgoDB"
	"fileStorage/params"
	"flag"
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/mux"
	"github.com/h2non/filetype"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const MB = 1 << 20
const EmptyRef = "00000000-0000-0000-0000-000000000000"

func main() {

	connectionString := flag.String("connectionString", "mongodb://sa:nokia@10.75.40.12:27017", "database server connection string")
	folderUpload := flag.String("folderUpload", "C:/Users/SmirnovA/PhpstormProjects/backend/uploads/refund/", "Путь к папке, где будут храниться файлы")
	folderStorage := flag.String("folderStorage", "E:/file/", "Путь к папке с хранилищем от 1С")
	listen := flag.String("listen", ":8585", "address and port")
	flag.Parse()

	mgoDB.NewConnectDB(*connectionString)

	params.SetFolderUpload(*folderUpload)
	params.SetFolderStorage(*folderStorage)

	router := mux.NewRouter()
	router.HandleFunc("/go/file/", uploadFile).Methods("POST")
	router.HandleFunc("/go/file/delete/", deleteFile).Methods("GET")
	router.HandleFunc("/go/file/{id}", getFile).Methods("GET")
	router.HandleFunc("/", indexPage)

	log.Println("Файловый сервис запущен. Папка хранения файлов " + params.GetFolderUpload() + " Порт " + *listen + " Папка хранилища " + params.GetFolderStorage())

	log.Fatal(http.ListenAndServe(*listen, router))

}

func indexPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

type FileUpload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Ext  string `json:"ext"`
	Size int64  `json:"size"`
	MIME string `json:"mime"`
}

func uploadFile(w http.ResponseWriter, r *http.Request) {

	r.Body = http.MaxBytesReader(w, r.Body, 10*MB)

	m, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "Ошибка чтения MultipartReader "+err.Error(), 400)
		return
	}

	filesInfo := []FileUpload{}

	for {

		p, err := m.NextPart()
		if err == io.EOF {
			break
		}

		if p.FileName() == "" {
			continue
		}

		fileID := newGUID()
		pathFile := params.GetFolderUpload() + fileID

		fileSrv, err := os.Create(pathFile)
		if err != nil {
			http.Error(w, "Ошибка создания файла "+err.Error(), 400)
			return
		}
		defer fileSrv.Close()

		_, err = io.Copy(fileSrv, p)
		if err != nil {
			http.Error(w, "Ошибка записи файла "+err.Error(), 400)
			return
		}

		fileOpen, err := os.Open(pathFile)
		if err != nil {
			http.Error(w, "Ошибка чтения файла "+err.Error(), 400)
			return
		}
		defer fileOpen.Close()

		kind, err := filetype.MatchReader(fileOpen)
		if err != nil {
			_ = os.Remove(pathFile)
			http.Error(w, "Ошибка получения информации о файле "+err.Error(), 400)
			return
		}

		if kind == filetype.Unknown {
			_ = os.Remove(pathFile)
			http.Error(w, "Неизвестный тип файла ", 400)
			return
		}

		fi, err := fileOpen.Stat()

		if kind.MIME.Value == "image/jpeg" ||
			kind.MIME.Value == "image/png" ||
			kind.MIME.Value == "image/gif" ||
			kind.MIME.Value == "image/bmp" ||
			kind.MIME.Value == "application/msword" ||
			kind.MIME.Value == "application/vnd.ms-excel" ||
			kind.MIME.Value == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" ||
			kind.MIME.Value == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" ||
			kind.MIME.Value == "application/vnd.ms-powerpoint" ||
			kind.MIME.Value == "application/pdf" {
		} else {
			//_ = os.Remove(pathFile)
			log.Println("UploadFiles: Тип файла " + kind.MIME.Value + " не доступен для загрузки " + pathFile)
			http.Error(w, "Тип файла "+kind.MIME.Value+" не доступен для загрузки", 400)
			return
		}

		fileInfo := FileUpload{
			ID:   fileID,
			Name: p.FileName(),
			Size: fi.Size(),
			MIME: kind.MIME.Value,
			Ext:  kind.Extension,
		}

		fileSrv.Close()
		fileOpen.Close()
		err = os.Rename(pathFile, pathFile+"."+kind.Extension)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		filesInfo = append(filesInfo, fileInfo)

	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filesInfo)

}

func newGUID() string {

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}

	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	return strings.ToUpper(uuid)
}

type FileAttach struct {
	ID   string    `bson:"_id"`
	Ext  string    `bson:"extension"`
	Date time.Time `bson:"dateCreation"`
}

func NewFileAttach() *FileAttach {
	return &FileAttach{
		ID: EmptyRef,
	}
}

func IsGuid(s string) bool {
	if len(s) != 36 {
		return false
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	for _, sub := range [...]string{
		s[0:8], s[9:13], s[14:18], s[19:23], s[24:36],
	} {
		for i := 0; i < len(sub); i++ {
			if hextable[sub[i]] == 16 {
				return false
			}
		}
	}
	return true
}

func getFile(w http.ResponseWriter, r *http.Request) {

	getParams := mux.Vars(r)
	fileId := getParams["id"]

	extIndex := strings.LastIndex(fileId, ".")
	if -1 != extIndex {
		fileId = fileId[:extIndex]
	}

	if !IsGuid(fileId) {
		log.Println("GetFile: неверный идентификатор файла " + fileId)
		http.Error(w, "Неверный идентификатор файла", 400)
		return
	}

	db := mgoDB.GetConnectDB().Copy()
	defer db.Close()

	fileAttach := NewFileAttach()
	selector := bson.M{"fileId": strings.ToUpper(fileId)}
	_ = db.DB("priceService").C("attachments").Find(selector).One(&fileAttach)

	if EmptyRef == fileAttach.ID {
		log.Println("GetFile: файл " + fileId + " не найден в БД")
		http.Error(w, "Файл не найден в БД", 404)
		return
	}

	path := params.GetFolderStorage()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Println("GetFile: каталог с файлами " + path + " недоступен")
		http.Error(w, "Ошибка чтения каталога", 400)
		return
	}

	MST, _ := time.LoadLocation("Europe/Moscow")
	date := fileAttach.Date.In(MST).Format("2006.01.02")

	filePath := path + date + "\\" + strings.ToLower(fileId) + "." + strings.ToLower(fileAttach.Ext)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Println("GetFile: файл " + filePath + " не найден")
		http.Error(w, "Файл не найден", 404)
		return
	}

	http.ServeFile(w, r, filePath)

}

var hextable = [...]byte{
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
	0x08, 0x09, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
	0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
}

func deleteFile(w http.ResponseWriter, r *http.Request) {

	path := params.GetFolderUpload()
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Println("Ошибка открытия каталога с файлами " + path + ": " + err.Error())
	}

	idFiles := make([]string, len(files))

	for index, file := range files {
		s := strings.Split(file.Name(), ".")
		idFiles[index] = s[0]
	}

	var idFilesNotDelete []string

	db := mgoDB.GetConnectDB().Copy()
	defer db.Close()

	selector := bson.M{"$and": []bson.M{{"send": true}, {"fileId": bson.M{"$in": idFiles}}}}
	iterator := db.DB("priceService").C("attachments").Find(selector).Iter()
	data := bson.M{}
	for iterator.Next(&data) {
		idFilesNotDelete = append(idFilesNotDelete, data["fileId"].(string))
	}

	now := time.Now().AddDate(0, 0, -2)

	for _, file := range files {

		s := strings.Split(file.Name(), ".")

		find := false
		for _, value := range idFilesNotDelete {
			if s[0] == value {
				find = true
			}
		}

		if find {

			log.Println("Файл " + file.Name() + " еще не отправлен")

		} else {

			if now.After(file.ModTime()) {
				log.Println("Удаляем. Дата создания", file.ModTime(), "меньше", now, file.Name())
			} else {
				log.Println("НЕ Удаляем. Дата создания", file.ModTime(), "больше", now, file.Name())
			}

		}
	}

}
