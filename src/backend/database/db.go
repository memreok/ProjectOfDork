package database

import (
	"dork-project/models"
	"log"
	"os"
	"strconv"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error

	dsn := os.Getenv("DATABASE_URL")
	dsn = strings.Trim(dsn, "'")
	dsn = strings.Trim(dsn, "\"")

	if dsn == "" {
		handleInitError("DATABASE_URL bulunamadı. Geçmiş özelliği devre dışı.")
		return
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		DB = nil
		handleInitError("Veritabanı bağlantı hatası: " + err.Error())
		return
	}

	err = db.AutoMigrate(&models.HistoryItem{})
	if err != nil {
		DB = nil
		handleInitError("Tablo oluşturma hatası: " + err.Error())
		return
	}

	DB = db
}

func IsReady() bool {
	return DB != nil
}

func IsRequired() bool {
	required, err := strconv.ParseBool(os.Getenv("DB_REQUIRED"))
	if err != nil {
		return false
	}
	return required
}

func handleInitError(message string) {
	if IsRequired() {
		log.Fatal(message)
	}
	log.Println(message)
}
