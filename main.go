package main

import (
	"crypto/rand"
	"encoding/json"
	"fileStorage/params"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/h2non/filetype"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	MB = 1 << 20
	//FOLDERUPLOAD = "C:/Users/SmirnovA/PhpstormProjects/backend/uploads/refund/"
	//FOLDERUPLOAD = "/var/www/Favorit/uploads/refund/"
)

func main() {

	folderUpload := flag.String("folderUpload", "C:/Users/SmirnovA/PhpstormProjects/backend/uploads/refund/", "Путь к папке, где будут храниться файлы")
	listen := flag.String("listen", ":8585", "address and port")
	flag.Parse()

	params.SetFolderUpload(*folderUpload)

	router := mux.NewRouter()
	router.HandleFunc("/go/file/", uploadFile).Methods("POST")
	router.HandleFunc("/", indexPage)

	log.Println("Файловый сервис запущен. Папка хранения файлов " + params.GetFolderUpload() + " Порт " + *listen)

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

/*func getFiles(w http.ResponseWriter, r *http.Request) {

	files, err := ioutil.ReadDir("\\\\domain\\corp\\1C\\1CBase\\WorkTrade\\ЗаявкаПокупателяНаВозврат\\")

	if err != nil {
		http.Error(w, "Ошибка чтения директории " + err.Error(), 400)
		return
	}

	dir := make([]string, 0)
	for _, file := range files {
		if file.IsDir() {
			dir = append(dir, file.Name())
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dir)

}*/
