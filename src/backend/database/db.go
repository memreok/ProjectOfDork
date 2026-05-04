package database

import (
	"dork-project/models"
	"log"
	"os"
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
		log.Fatal("DATABASE_URL bulunamadı. Lütfen .env dosyasını kontrol edin.")
	}

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Veritabanı bağlantı hatası: ", err)
	}

	err = DB.AutoMigrate(&models.HistoryItem{})
	if err != nil {
		log.Fatal("Tablo oluşturma hatası: ", err)
	}
}
