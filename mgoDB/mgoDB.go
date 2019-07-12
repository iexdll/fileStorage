package mgoDB

import (
	"github.com/globalsign/mgo"
	"log"
)

var mgoSession *mgo.Session

func NewConnectDB(connectionString string) {

	var err error

	mgoSession, err = mgo.Dial(connectionString)
	if err != nil {
		log.Fatal("Ошибка подключения к MongoDB " + connectionString)
	} else {
		log.Println("Успешно подключен к MongoDB " + connectionString)
	}
}

func GetConnectDB() *mgo.Session {
	return mgoSession
}
