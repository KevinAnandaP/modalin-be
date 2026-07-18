package database

import (
	"log"
	"time"

	"modalin-be/internal/model"

	"golang.org/x/crypto/bcrypt"
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

	// 3. Seed Default Admin User
	adminEmail := "admin@modalin.id"
	var existingAdmin model.User
	err := db.Where("email = ?", adminEmail).First(&existingAdmin).Error
	if err == gorm.ErrRecordNotFound {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin12345"), bcrypt.DefaultCost)
		if err == nil {
			now := time.Now().UTC()
			adminUser := model.User{
				FullName:        "System Administrator",
				Email:           adminEmail,
				Phone:           "080000000000",
				PasswordHash:    string(hash),
				City:            "Jakarta",
				Address:         "Modalin HQ",
				Status:          "active",
				TermsAcceptedAt: &now,
				TermsVersion:    "v1",
			}
			if err := db.Create(&adminUser).Error; err == nil {
				log.Printf("Seeded default admin user: %s", adminEmail)
				userRole := model.UserRole{
					UserID:     adminUser.ID,
					RoleID:     4, // Admin role
					Status:     "approved",
					ApprovedAt: &now,
				}
				db.Create(&userRole)
			}
		}
	} else if err == nil {
		var existingUserRole model.UserRole
		errUR := db.Where("user_id = ? AND role_id = ?", existingAdmin.ID, 4).First(&existingUserRole).Error
		if errUR == gorm.ErrRecordNotFound {
			now := time.Now().UTC()
			userRole := model.UserRole{
				UserID:     existingAdmin.ID,
				RoleID:     4,
				Status:     "approved",
				ApprovedAt: &now,
			}
			db.Create(&userRole)
		}
	}

	log.Println("Database seeding completed.")
}
