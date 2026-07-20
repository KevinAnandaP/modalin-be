package database

import (
	"log"
	"modalin-be/internal/model"

	"gorm.io/gorm"
)

func SeedData(db *gorm.DB) {
	log.Println("Starting database seeding...")

	// 1. Seed Roles
	roles := []model.Role{
		{ID: 1, Name: "borrower", Description: "Peminjam Modal (UMKM)"},
		{ID: 2, Name: "lender", Description: "Pemberi Modal (Investor)"},
		{ID: 3, Name: "verifier", Description: "Verifikator Usaha"},
		{ID: 4, Name: "admin", Description: "Administrator Sistem"},
	}

	for _, r := range roles {
		var existing model.Role
		err := db.First(&existing, r.ID).Error
		if err == gorm.ErrRecordNotFound {
			if err := db.Create(&r).Error; err != nil {
				log.Printf("Failed to seed role %s: %v", r.Name, err)
			} else {
				log.Printf("Seeded role: %s", r.Name)
			}
		} else if err != nil {
			log.Printf("Error checking role %d: %v", r.ID, err)
		}
	}

	// 2. Seed Business Categories
	desc1 := "Bisnis makanan, minuman, katering, franchise kuliner, dll."
	desc2 := "Penyedia jasa/layanan seperti laundry, bengkel, cukur, salon, dll."
	desc3 := "Budidaya tanaman, perkebunan, hortikultura, pangan, dll."
	desc4 := "Kerajinan tangan, souvenir, anyaman, produk kreatif, dll."
	desc5 := "Toko kelontong, retail, reseller, e-commerce, keagenan, dll."
	desc6 := "Penyedia solusi IT, agensi web/app, IoT lokal, dll."
	desc7 := "Kategori bisnis lainnya."

	categories := []model.BusinessCategory{
		{ID: 1, Name: "Kuliner", Description: &desc1},
		{ID: 2, Name: "Jasa", Description: &desc2},
		{ID: 3, Name: "Pertanian", Description: &desc3},
		{ID: 4, Name: "Kerajinan", Description: &desc4},
		{ID: 5, Name: "Perdagangan", Description: &desc5},
		{ID: 6, Name: "Teknologi", Description: &desc6},
		{ID: 7, Name: "Lainnya", Description: &desc7},
	}

	for _, c := range categories {
		var existing model.BusinessCategory
		err := db.First(&existing, c.ID).Error
		if err == gorm.ErrRecordNotFound {
			if err := db.Create(&c).Error; err != nil {
				log.Printf("Failed to seed category %s: %v", c.Name, err)
			} else {
				log.Printf("Seeded category: %s", c.Name)
			}
		} else if err != nil {
			log.Printf("Error checking category %d: %v", c.ID, err)
		}
	}

	log.Println("Database seeding completed.")
}
