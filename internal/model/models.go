package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 1. User
type User struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	FullName        string         `gorm:"type:varchar(255);not null"`
	Email           string         `gorm:"type:varchar(255);uniqueIndex;not null"`
	Phone           string         `gorm:"type:varchar(50);uniqueIndex;not null"`
	PasswordHash    string         `gorm:"type:varchar(255);not null" json:"-"`
	City            string         `gorm:"type:varchar(100);not null"`
	Address         string         `gorm:"type:text;not null"`
	Status          string         `gorm:"type:varchar(50);default:'active';not null"` // active, suspended, blocked
	GoogleID        *string        `gorm:"type:varchar(255);uniqueIndex"`
	TermsAcceptedAt *time.Time     `gorm:"type:timestamp"`
	TermsVersion    string         `gorm:"type:varchar(50)"`
	CreatedAt       time.Time      `gorm:"not null"`
	UpdatedAt       time.Time      `gorm:"not null"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	// Relations
	Roles    []UserRole `gorm:"foreignKey:UserID"`
	Business *Business  `gorm:"foreignKey:UserID"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}

// 2. Role
type Role struct {
	ID          int    `gorm:"primaryKey"`
	Name        string `gorm:"type:varchar(50);uniqueIndex;not null"` // borrower, lender, verifier, admin
	Description string `gorm:"type:text;not null"`
}

func (Role) TableName() string {
	return "roles"
}

// 3. UserRole
type UserRole struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     uuid.UUID      `gorm:"type:uuid;index;not null"`
	RoleID     int            `gorm:"index;not null"`
	Status     string         `gorm:"type:varchar(50);default:'pending';not null"` // pending, approved, rejected, revoked
	ApprovedBy *uuid.UUID     `gorm:"type:uuid;index"`
	ApprovedAt *time.Time     `gorm:"type:timestamp"`
	CreatedAt  time.Time      `gorm:"not null"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`

	// Relations
	User     User  `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Role     Role  `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Approver *User `gorm:"foreignKey:ApprovedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

// RoleRequest represents the approval workflow; it is separate from active roles.
type RoleRequest struct {
	ID                    uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID                uuid.UUID      `gorm:"type:uuid;index;not null"`
	RoleID                int            `gorm:"index;not null"`
	Status                string         `gorm:"type:varchar(50);default:'submitted';not null"` // submitted, under_review, revision_required, approved, rejected
	IdentityCardURL       *string        `gorm:"type:text"`                                     // Foto KTP / Identitas
	RiskAgreementAccepted bool           `gorm:"type:boolean;default:false;not null"`           // Khusus Lender
	EthicsAccepted        bool           `gorm:"type:boolean;default:false;not null"`           // Khusus Verifier
	TrainingCompleted     bool           `gorm:"type:boolean;default:false;not null"`           // Khusus Verifier
	AdminNote             *string        `gorm:"type:text"`                                     // Catatan admin saat revisi/penolakan
	ApprovedBy            *uuid.UUID     `gorm:"type:uuid;index"`
	ApprovedAt            *time.Time     `gorm:"type:timestamp"`
	CreatedAt             time.Time      `gorm:"not null"`
	UpdatedAt             time.Time      `gorm:"not null"`
	DeletedAt             gorm.DeletedAt `gorm:"index"`

	User     User  `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Role     Role  `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Approver *User `gorm:"foreignKey:ApprovedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

func (RoleRequest) TableName() string { return "role_requests" }

func (rr *RoleRequest) BeforeCreate(tx *gorm.DB) (err error) {
	if rr.ID == uuid.Nil {
		rr.ID = uuid.New()
	}
	return nil
}

func (UserRole) TableName() string {
	return "user_roles"
}

func (ur *UserRole) BeforeCreate(tx *gorm.DB) (err error) {
	if ur.ID == uuid.Nil {
		ur.ID = uuid.New()
	}
	return
}

// 4. BusinessCategory
type BusinessCategory struct {
	ID          int     `gorm:"primaryKey"`
	Name        string  `gorm:"type:varchar(100);uniqueIndex;not null"`
	Description *string `gorm:"type:text"`
}

func (BusinessCategory) TableName() string {
	return "business_categories"
}

// 5. Business
type Business struct {
	ID                    uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID                uuid.UUID      `gorm:"type:uuid;index;not null"`
	BusinessName          string         `gorm:"type:varchar(255);not null"`
	CategoryID            int            `gorm:"index;not null"`
	Description           string         `gorm:"type:text;not null"`
	BusinessType          string         `gorm:"type:varchar(50);not null"`                  // running, starter
	Status                string         `gorm:"type:varchar(50);default:'active';not null"` // active, inactive
	StartedAt             *time.Time     `gorm:"type:date"`
	LocationAddress       string         `gorm:"type:text;not null"`
	Latitude              *float64       `gorm:"type:decimal(10,8)"`
	Longitude             *float64       `gorm:"type:decimal(11,8)"`
	PhotoURL              *string        `gorm:"type:text"`
	VerificationStatus    string         `gorm:"type:varchar(50);default:'unverified';not null"` // unverified, pending, verified, rejected
	TrustScore            int            `gorm:"type:int;default:0;not null"`
	CurrentBorrowingLimit int64          `gorm:"type:bigint;default:300000;not null"` // limit pinjaman aktif (misal default 300k untuk rintisan)
	CreatedAt             time.Time      `gorm:"not null"`
	UpdatedAt             time.Time      `gorm:"not null"`
	DeletedAt             gorm.DeletedAt `gorm:"index"`

	// Relations
	User             User                   `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Category         BusinessCategory       `gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	FinancialRecords []FinancialRecord      `gorm:"foreignKey:BusinessID"`
	Campaigns        []LoanCampaign         `gorm:"foreignKey:BusinessID"`
	StarterDetails   *StarterBusinessDetail `gorm:"foreignKey:BusinessID"`
}

func (Business) TableName() string {
	return "businesses"
}

func (b *Business) BeforeCreate(tx *gorm.DB) (err error) {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return
}

// 6. FinancialRecord
type FinancialRecord struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BusinessID    uuid.UUID      `gorm:"type:uuid;index;not null"`
	RecordDate    time.Time      `gorm:"type:date;not null"`
	IncomeAmount  int64          `gorm:"type:bigint;not null"`
	ExpenseAmount int64          `gorm:"type:bigint;not null"`
	NetAmount     int64          `gorm:"type:bigint;not null"` // income_amount - expense_amount
	Note          *string        `gorm:"type:text"`
	Status        string         `gorm:"type:varchar(50);default:'draft';not null"` // draft, submitted, verified, rejected
	VerifiedBy    *uuid.UUID     `gorm:"type:uuid;index"`
	CreatedAt     time.Time      `gorm:"not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`

	// Relations
	Business Business               `gorm:"foreignKey:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Verifier *User                  `gorm:"foreignKey:VerifiedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Proofs   []FinancialRecordProof `gorm:"foreignKey:FinancialRecordID"`
}

func (FinancialRecord) TableName() string {
	return "financial_records"
}

func (fr *FinancialRecord) BeforeCreate(tx *gorm.DB) (err error) {
	if fr.ID == uuid.Nil {
		fr.ID = uuid.New()
	}
	return
}

// 7. FinancialRecordProof
type FinancialRecordProof struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	FinancialRecordID uuid.UUID      `gorm:"type:uuid;index;not null"`
	FileURL           string         `gorm:"type:text;not null"`
	ProofType         string         `gorm:"type:varchar(50);not null"` // receipt, invoice, photo, transfer, other
	Amount            *int64         `gorm:"type:bigint"`
	Status            string         `gorm:"type:varchar(50);default:'pending';not null"` // pending, approved, rejected
	CreatedAt         time.Time      `gorm:"not null"`
	DeletedAt         gorm.DeletedAt `gorm:"index"`

	// Relations
	FinancialRecord FinancialRecord `gorm:"foreignKey:FinancialRecordID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (FinancialRecordProof) TableName() string {
	return "financial_record_proofs"
}

func (frp *FinancialRecordProof) BeforeCreate(tx *gorm.DB) (err error) {
	if frp.ID == uuid.Nil {
		frp.ID = uuid.New()
	}
	return
}

// 8. VerificationRequest
type VerificationRequest struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BusinessID  uuid.UUID      `gorm:"type:uuid;index;not null"`
	RequestedBy uuid.UUID      `gorm:"type:uuid;index;not null"`
	Status      string         `gorm:"type:varchar(50);default:'pending';not null"` // pending, assigned, reviewed, approved, rejected
	AdminNote   *string        `gorm:"type:text"`
	CreatedAt   time.Time      `gorm:"not null"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	// Relations
	Business  Business             `gorm:"foreignKey:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Requester User                 `gorm:"foreignKey:RequestedBy;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Reports   []VerificationReport `gorm:"foreignKey:VerificationRequestID"`
}

func (VerificationRequest) TableName() string {
	return "verification_requests"
}

func (vr *VerificationRequest) BeforeCreate(tx *gorm.DB) (err error) {
	if vr.ID == uuid.Nil {
		vr.ID = uuid.New()
	}
	return
}

// 9. VerificationReport
type VerificationReport struct {
	ID                    uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VerificationRequestID uuid.UUID      `gorm:"type:uuid;index;not null"`
	VerifierUserID        uuid.UUID      `gorm:"type:uuid;index;not null"`
	IsBusinessExists      bool           `gorm:"type:boolean;not null"`
	IsBusinessActive      bool           `gorm:"type:boolean;not null"`
	LocationMatch         bool           `gorm:"type:boolean;not null"`
	PhotoURL              string         `gorm:"type:text;not null"`
	Note                  string         `gorm:"type:text;not null"`
	Recommendation        string         `gorm:"type:varchar(50);not null"` // approve, review, reject
	CreatedAt             time.Time      `gorm:"not null"`
	DeletedAt             gorm.DeletedAt `gorm:"index"`

	// Relations
	VerificationRequest VerificationRequest `gorm:"foreignKey:VerificationRequestID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Verifier            User                `gorm:"foreignKey:VerifierUserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}

func (VerificationReport) TableName() string {
	return "verification_reports"
}

func (vrp *VerificationReport) BeforeCreate(tx *gorm.DB) (err error) {
	if vrp.ID == uuid.Nil {
		vrp.ID = uuid.New()
	}
	return
}

// 10. CommunityVote
type CommunityVote struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BusinessID uuid.UUID      `gorm:"type:uuid;index;not null"`
	UserID     uuid.UUID      `gorm:"type:uuid;index;not null"`
	VoteType   string         `gorm:"type:varchar(50);not null"` // up, down
	Reason     *string        `gorm:"type:text"`
	CreatedAt  time.Time      `gorm:"not null"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`

	// Relations
	Business Business `gorm:"foreignKey:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	User     User     `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}

func (CommunityVote) TableName() string {
	return "community_votes"
}

func (cv *CommunityVote) BeforeCreate(tx *gorm.DB) (err error) {
	if cv.ID == uuid.Nil {
		cv.ID = uuid.New()
	}
	return
}

// 11. LoanCampaign
type LoanCampaign struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BusinessID          uuid.UUID      `gorm:"type:uuid;index;not null"`
	Title               string         `gorm:"type:varchar(255);not null"`
	Description         string         `gorm:"type:text;not null"`
	RequestedAmount     int64          `gorm:"type:bigint;not null"`
	FundedAmount        int64          `gorm:"type:bigint;default:0;not null"`
	LoanTenorMonths     int            `gorm:"type:int;not null"`
	BenefitType         string         `gorm:"type:varchar(50);not null"` // fixed_margin, revenue_share, principal_only
	MarginPercent       *float64       `gorm:"type:decimal(5,2)"`
	RevenueSharePercent *float64       `gorm:"type:decimal(5,2)"`
	ReturnCapPercent    *float64       `gorm:"type:decimal(5,2)"`
	RiskLevel           string         `gorm:"type:varchar(50);default:'medium';not null"` // low, medium, high
	Status              string         `gorm:"type:varchar(50);default:'draft';not null"`  // draft, admin_review, published, funded, active, completed, defaulted, rejected
	ApprovedBy          *uuid.UUID     `gorm:"type:uuid;index"`
	ApprovedAt          *time.Time     `gorm:"type:timestamp"`
	CreatedAt           time.Time      `gorm:"not null"`
	DeletedAt           gorm.DeletedAt `gorm:"index"`

	// Relations
	Business              Business                        `gorm:"foreignKey:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Approver              *User                           `gorm:"foreignKey:ApprovedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	BudgetItems           []CampaignBudgetItem            `gorm:"foreignKey:CampaignID"`
	Milestones            []CampaignMilestone             `gorm:"foreignKey:CampaignID"`
	Fundings              []Funding                       `gorm:"foreignKey:CampaignID"`
	Disbursements         []Disbursement                  `gorm:"foreignKey:CampaignID"`
	RevenueReports        []RevenueReport                 `gorm:"foreignKey:CampaignID"`
	Schedules             []RepaymentSchedule             `gorm:"foreignKey:CampaignID"`
	Repayments            []Repayment                     `gorm:"foreignKey:CampaignID"`
	RestructuringRequests []RepaymentRestructuringRequest `gorm:"foreignKey:CampaignID"`
}

func (LoanCampaign) TableName() string {
	return "loan_campaigns"
}

func (lc *LoanCampaign) BeforeCreate(tx *gorm.DB) (err error) {
	if lc.ID == uuid.Nil {
		lc.ID = uuid.New()
	}
	return
}

// 12. CampaignBudgetItem
type CampaignBudgetItem struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CampaignID     uuid.UUID `gorm:"type:uuid;index;not null"`
	ItemName       string    `gorm:"type:varchar(255);not null"`
	Category       string    `gorm:"type:varchar(50);not null"` // equipment, stock, packaging, transport, marketing, other
	Amount         int64     `gorm:"type:bigint;not null"`
	PriorityLevel  string    `gorm:"type:varchar(10)"`            // high, medium, low; legacy items remain NULL until revised
	EntryOrder     int       `gorm:"type:int;default:0;not null"` // stable tie-breaker when priorities match
	PurchaseMethod string    `gorm:"type:varchar(50);not null"`   // direct_purchase, cash_limited
	Note           *string   `gorm:"type:text"`

	// Relations
	Campaign LoanCampaign `gorm:"foreignKey:CampaignID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (CampaignBudgetItem) TableName() string {
	return "campaign_budget_items"
}

func (cbi *CampaignBudgetItem) BeforeCreate(tx *gorm.DB) (err error) {
	if cbi.ID == uuid.Nil {
		cbi.ID = uuid.New()
	}
	return
}

// 13. CampaignMilestone
type CampaignMilestone struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CampaignID   uuid.UUID      `gorm:"type:uuid;index;not null"`
	BudgetItemID *uuid.UUID     `gorm:"type:uuid;uniqueIndex"` // nullable only for legacy manually-created milestones
	Title        string         `gorm:"type:varchar(255);not null"`
	Amount       int64          `gorm:"type:bigint;not null"`
	SequenceNo   int            `gorm:"type:int;not null"`
	Status       string         `gorm:"type:varchar(50);default:'locked';not null"` // locked, available, disbursed, verified, rejected
	DueDate      *time.Time     `gorm:"type:date"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	// Relations
	Campaign   LoanCampaign       `gorm:"foreignKey:CampaignID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	BudgetItem CampaignBudgetItem `gorm:"foreignKey:BudgetItemID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}

func (CampaignMilestone) TableName() string {
	return "campaign_milestones"
}

func (cm *CampaignMilestone) BeforeCreate(tx *gorm.DB) (err error) {
	if cm.ID == uuid.Nil {
		cm.ID = uuid.New()
	}
	return
}

// 14. Funding
type Funding struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CampaignID   uuid.UUID      `gorm:"type:uuid;index;not null"`
	LenderUserID uuid.UUID      `gorm:"type:uuid;index;not null"`
	Amount       int64          `gorm:"type:bigint;not null"`
	Status       string         `gorm:"type:varchar(50);default:'pledged';not null"` // pledged, paid, cancelled, refunded
	FundedAt     time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	// Relations
	Campaign LoanCampaign `gorm:"foreignKey:CampaignID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Lender   User         `gorm:"foreignKey:LenderUserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}

func (Funding) TableName() string {
	return "fundings"
}

func (f *Funding) BeforeCreate(tx *gorm.DB) (err error) {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return
}

// 15. Disbursement
type Disbursement struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CampaignID    uuid.UUID      `gorm:"type:uuid;index;not null"`
	MilestoneID   uuid.UUID      `gorm:"type:uuid;index;not null"`
	Amount        int64          `gorm:"type:bigint;not null"`
	Method        string         `gorm:"type:varchar(50);not null"` // direct_purchase, cash_limited, bank_transfer, ewallet
	RecipientType string         `gorm:"type:varchar(50);not null"` // borrower, merchant
	RecipientName *string        `gorm:"type:varchar(255)"`
	Status        string         `gorm:"type:varchar(50);default:'pending';not null"` // pending, released, proof_required, verified, rejected
	ReleasedAt    *time.Time     `gorm:"type:timestamp"`
	CreatedAt     time.Time      `gorm:"not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`

	// Relations
	Campaign  LoanCampaign      `gorm:"foreignKey:CampaignID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Milestone CampaignMilestone `gorm:"foreignKey:MilestoneID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}

func (Disbursement) TableName() string {
	return "disbursements"
}

func (d *Disbursement) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return
}

// 16. FundUsageProof
type FundUsageProof struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DisbursementID uuid.UUID      `gorm:"type:uuid;index;not null"`
	CampaignID     uuid.UUID      `gorm:"type:uuid;index;not null"`
	FileURL        string         `gorm:"type:text;not null"`
	ProofType      string         `gorm:"type:varchar(50);not null"` // invoice, receipt, photo, merchant_confirmation
	Amount         int64          `gorm:"type:bigint;not null"`
	Note           *string        `gorm:"type:text"`
	Status         string         `gorm:"type:varchar(50);default:'pending';not null"` // pending, approved, rejected, revision_needed
	ReviewedBy     *uuid.UUID     `gorm:"type:uuid;index"`
	ReviewedAt     *time.Time     `gorm:"type:timestamp"`
	CreatedAt      time.Time      `gorm:"not null"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`

	// Relations
	Disbursement Disbursement `gorm:"foreignKey:DisbursementID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Campaign     LoanCampaign `gorm:"foreignKey:CampaignID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Reviewer     *User        `gorm:"foreignKey:ReviewedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

func (FundUsageProof) TableName() string {
	return "fund_usage_proofs"
}

func (fup *FundUsageProof) BeforeCreate(tx *gorm.DB) (err error) {
	if fup.ID == uuid.Nil {
		fup.ID = uuid.New()
	}
	return
}

// 17. RevenueReport
type RevenueReport struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CampaignID      uuid.UUID      `gorm:"type:uuid;index;not null"`
	BusinessID      uuid.UUID      `gorm:"type:uuid;index;not null"`
	PeriodMonth     int            `gorm:"type:int;not null"`
	PeriodYear      int            `gorm:"type:int;not null"`
	GrossRevenue    int64          `gorm:"type:bigint;not null"`
	VerifiedRevenue *int64         `gorm:"type:bigint"`
	ExpenseTotal    *int64         `gorm:"type:bigint"`
	Note            *string        `gorm:"type:text"`
	Status          string         `gorm:"type:varchar(50);default:'draft';not null"` // draft, submitted, verified, rejected, under_review
	VerifiedBy      *uuid.UUID     `gorm:"type:uuid;index"`
	CreatedAt       time.Time      `gorm:"not null"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	// Relations
	Campaign LoanCampaign         `gorm:"foreignKey:CampaignID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Business Business             `gorm:"foreignKey:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Verifier *User                `gorm:"foreignKey:VerifiedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Proofs   []RevenueReportProof `gorm:"foreignKey:RevenueReportID"`
}

func (RevenueReport) TableName() string {
	return "revenue_reports"
}

func (rr *RevenueReport) BeforeCreate(tx *gorm.DB) (err error) {
	if rr.ID == uuid.Nil {
		rr.ID = uuid.New()
	}
	return
}

// 18. RevenueReportProof
type RevenueReportProof struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RevenueReportID uuid.UUID      `gorm:"type:uuid;index;not null"`
	FileURL         string         `gorm:"type:text;not null"`
	ProofType       string         `gorm:"type:varchar(50);not null"` // receipt, invoice, sales_recap, bank_statement, photo
	Amount          *int64         `gorm:"type:bigint"`
	Status          string         `gorm:"type:varchar(50);default:'pending';not null"` // pending, approved, rejected
	CreatedAt       time.Time      `gorm:"not null"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	// Relations
	RevenueReport RevenueReport `gorm:"foreignKey:RevenueReportID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (RevenueReportProof) TableName() string {
	return "revenue_report_proofs"
}

func (rrp *RevenueReportProof) BeforeCreate(tx *gorm.DB) (err error) {
	if rrp.ID == uuid.Nil {
		rrp.ID = uuid.New()
	}
	return
}

// 19. RepaymentSchedule
type RepaymentSchedule struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CampaignID      uuid.UUID      `gorm:"type:uuid;index;not null"`
	DueDate         time.Time      `gorm:"type:date;not null"`
	PrincipalDue    int64          `gorm:"type:bigint;not null"`
	MarginDue       *int64         `gorm:"type:bigint"`
	RevenueShareDue *int64         `gorm:"type:bigint"`
	TotalDue        int64          `gorm:"type:bigint;not null"`
	Status          string         `gorm:"type:varchar(50);default:'upcoming';not null"` // upcoming, due, paid, late, restructured
	CreatedAt       time.Time      `gorm:"not null"`
	UpdatedAt       time.Time      `gorm:"not null"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	// Relations
	Campaign LoanCampaign `gorm:"foreignKey:CampaignID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}

func (RepaymentSchedule) TableName() string {
	return "repayment_schedules"
}

func (rs *RepaymentSchedule) BeforeCreate(tx *gorm.DB) (err error) {
	if rs.ID == uuid.Nil {
		rs.ID = uuid.New()
	}
	return
}

// 20. Repayment
type Repayment struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CampaignID      uuid.UUID      `gorm:"type:uuid;index;not null"`
	ScheduleID      uuid.UUID      `gorm:"type:uuid;index;not null"`
	PaidAmount      int64          `gorm:"type:bigint;not null"`
	PrincipalPaid   int64          `gorm:"type:bigint;not null"`
	BenefitPaid     int64          `gorm:"type:bigint;not null"`
	PaymentProofURL *string        `gorm:"type:text"`
	Status          string         `gorm:"type:varchar(50);default:'pending';not null"` // pending, verified, rejected
	PaidAt          time.Time      `gorm:"not null"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	// Relations
	Campaign LoanCampaign      `gorm:"foreignKey:CampaignID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Schedule RepaymentSchedule `gorm:"foreignKey:ScheduleID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}

func (Repayment) TableName() string {
	return "repayments"
}

func (r *Repayment) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}

// 21. LenderReturnDistribution
type LenderReturnDistribution struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RepaymentID     uuid.UUID `gorm:"type:uuid;index;not null"`
	FundingID       uuid.UUID `gorm:"type:uuid;index;not null"`
	LenderUserID    uuid.UUID `gorm:"type:uuid;index;not null"`
	PrincipalAmount int64     `gorm:"type:bigint;not null"`
	BenefitAmount   int64     `gorm:"type:bigint;not null"`
	Status          string    `gorm:"type:varchar(50);default:'pending';not null"` // pending, distributed
	CreatedAt       time.Time `gorm:"not null"`

	// Relations
	Repayment Repayment `gorm:"foreignKey:RepaymentID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Funding   Funding   `gorm:"foreignKey:FundingID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Lender    User      `gorm:"foreignKey:LenderUserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}

func (LenderReturnDistribution) TableName() string {
	return "lender_return_distributions"
}

func (lrd *LenderReturnDistribution) BeforeCreate(tx *gorm.DB) (err error) {
	if lrd.ID == uuid.Nil {
		lrd.ID = uuid.New()
	}
	return
}

// 22. RiskAssessment
type RiskAssessment struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CampaignID        uuid.UUID      `gorm:"type:uuid;index;not null"`
	BusinessID        uuid.UUID      `gorm:"type:uuid;index;not null"`
	FinancialScore    int            `gorm:"type:int;not null"`
	VerificationScore int            `gorm:"type:int;not null"`
	RepaymentScore    int            `gorm:"type:int;not null"`
	CommunityScore    int            `gorm:"type:int;not null"`
	FinalScore        int            `gorm:"type:int;not null"`
	RiskLevel         string         `gorm:"type:varchar(50);not null"` // low, medium, high
	Note              *string        `gorm:"type:text"`
	CreatedAt         time.Time      `gorm:"not null"`
	DeletedAt         gorm.DeletedAt `gorm:"index"`

	// Relations
	Campaign LoanCampaign `gorm:"foreignKey:CampaignID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Business Business     `gorm:"foreignKey:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}

func (RiskAssessment) TableName() string {
	return "risk_assessments"
}

func (ra *RiskAssessment) BeforeCreate(tx *gorm.DB) (err error) {
	if ra.ID == uuid.Nil {
		ra.ID = uuid.New()
	}
	return
}

// 23. Dispute
type Dispute struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CampaignID   uuid.UUID      `gorm:"type:uuid;index;not null"`
	ReportedBy   uuid.UUID      `gorm:"type:uuid;index;not null"`
	TargetUserID *uuid.UUID     `gorm:"type:uuid;index"`
	Type         string         `gorm:"type:varchar(100);not null"` // fraud_suspected, late_payment, invalid_proof, misuse_of_funds, other
	Description  string         `gorm:"type:text;not null"`
	Status       string         `gorm:"type:varchar(50);default:'open';not null"` // open, under_review, resolved, rejected
	ResolvedBy   *uuid.UUID     `gorm:"type:uuid;index"`
	CreatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	// Relations
	Campaign LoanCampaign `gorm:"foreignKey:CampaignID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Reporter User         `gorm:"foreignKey:ReportedBy;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Target   *User        `gorm:"foreignKey:TargetUserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Resolver *User        `gorm:"foreignKey:ResolvedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

func (Dispute) TableName() string {
	return "disputes"
}

func (d *Dispute) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return
}

// 24. AuditLog
type AuditLog struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     *uuid.UUID `gorm:"type:uuid;index"`
	Action     string     `gorm:"type:varchar(255);not null"`
	EntityType string     `gorm:"type:varchar(100);not null"`
	EntityID   uuid.UUID  `gorm:"type:uuid;not null"`
	OldValue   *string    `gorm:"type:json"`
	NewValue   *string    `gorm:"type:json"`
	CreatedAt  time.Time  `gorm:"not null"`

	// Relations
	User *User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

func (al *AuditLog) BeforeCreate(tx *gorm.DB) (err error) {
	if al.ID == uuid.Nil {
		al.ID = uuid.New()
	}
	return
}

// 25. StarterBusinessDetail (Jalur khusus Modal Rintisan - 1-to-1 dengan Business)
type StarterBusinessDetail struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BusinessID        uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null"`
	TargetMarket      string         `gorm:"type:text;not null"`                  // Target pembeli awal
	SupplierInfo      string         `gorm:"type:text;not null"`                  // Sumber bahan baku / tempat beli alat
	PricingEstimation string         `gorm:"type:text;not null"`                  // Perkiraan harga modal dan jual
	ReadinessProofURL *string        `gorm:"type:text"`                           // URL bukti kesiapan (foto contoh produk, lokasi, chat pembeli, dll.)
	CommitmentChecked bool           `gorm:"type:boolean;default:false;not null"` // Komitmen melakukan pencatatan keuangan setelah dana cair
	CreatedAt         time.Time      `gorm:"not null"`
	UpdatedAt         time.Time      `gorm:"not null"`
	DeletedAt         gorm.DeletedAt `gorm:"index"`

	// Relations
	Business Business `gorm:"foreignKey:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (StarterBusinessDetail) TableName() string {
	return "starter_business_details"
}

func (sbd *StarterBusinessDetail) BeforeCreate(tx *gorm.DB) (err error) {
	if sbd.ID == uuid.Nil {
		sbd.ID = uuid.New()
	}
	return
}

// 26. RepaymentRestructuringRequest (Log pengajuan restrukturisasi pembayaran cicilan)
type RepaymentRestructuringRequest struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CampaignID          uuid.UUID      `gorm:"type:uuid;index;not null"`
	Reason              string         `gorm:"type:text;not null"`                          // Alasan pengajuan
	ProposedTenorMonths int            `gorm:"type:int;not null"`                           // Tenor baru yang diusulkan
	ProofURL            *string        `gorm:"type:text"`                                   // Foto / dokumen bukti kondisi usaha saat ini
	Status              string         `gorm:"type:varchar(50);default:'pending';not null"` // pending, approved, rejected
	AdminNote           *string        `gorm:"type:text"`                                   // Catatan tinjauan admin
	ApprovedBy          *uuid.UUID     `gorm:"type:uuid;index"`                             // Admin yang menyetujui/menolak
	ApprovedAt          *time.Time     `gorm:"type:timestamp"`                              // Waktu persetujuan
	CreatedAt           time.Time      `gorm:"not null"`
	UpdatedAt           time.Time      `gorm:"not null"`
	DeletedAt           gorm.DeletedAt `gorm:"index"`

	// Relations
	Campaign LoanCampaign `gorm:"foreignKey:CampaignID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Approver *User        `gorm:"foreignKey:ApprovedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

func (RepaymentRestructuringRequest) TableName() string {
	return "repayment_restructuring_requests"
}

func (rrr *RepaymentRestructuringRequest) BeforeCreate(tx *gorm.DB) (err error) {
	if rrr.ID == uuid.Nil {
		rrr.ID = uuid.New()
	}
	return
}
