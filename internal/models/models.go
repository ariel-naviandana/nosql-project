package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ========== NASABAH ==========

type Address struct {
	Street     string `bson:"street" json:"street"`
	City       string `bson:"city" json:"city"`
	Province   string `bson:"province" json:"province"`
	PostalCode string `bson:"postal_code" json:"postal_code"`
}

type Nasabah struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	AccountID   string             `bson:"account_id" json:"account_id"` // ref to PostgreSQL
	NIK         string             `bson:"nik" json:"nik"`
	NamaLengkap string             `bson:"nama_lengkap" json:"nama_lengkap"`
	Email       string             `bson:"email" json:"email"`
	Password    string             `bson:"password,omitempty" json:"-"`
	NoHP        string             `bson:"no_hp" json:"no_hp"`
	TanggalLahir string            `bson:"tanggal_lahir" json:"tanggal_lahir"`
	Alamat      Address            `bson:"alamat" json:"alamat"`
	StatusKYC   string             `bson:"status_kyc" json:"status_kyc"` // pending, verified, rejected
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// ========== DOKUMEN KYC ==========

type DokumenKYC struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	NasabahID  primitive.ObjectID `bson:"nasabah_id" json:"nasabah_id"`
	AccountID  string             `bson:"account_id" json:"account_id"`
	TipeDokumen string            `bson:"tipe_dokumen" json:"tipe_dokumen"` // ktp, selfie, npwp, signature
	NamaFile   string             `bson:"nama_file" json:"nama_file"`
	MimeType   string             `bson:"mime_type" json:"mime_type"`
	FileBase64 string             `bson:"file_base64,omitempty" json:"file_base64,omitempty"`
	FileURL    string             `bson:"file_url,omitempty" json:"file_url,omitempty"`
	Status     string             `bson:"status" json:"status"` // uploaded, verified, rejected
	Keterangan string             `bson:"keterangan,omitempty" json:"keterangan,omitempty"`
	UploadedAt time.Time          `bson:"uploaded_at" json:"uploaded_at"`
	VerifiedAt *time.Time         `bson:"verified_at,omitempty" json:"verified_at,omitempty"`
}

// ========== AUDIT LOG ==========

type AuditLog struct {
	ID         primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	AccountID  string                 `bson:"account_id" json:"account_id"`
	NasabahID  string                 `bson:"nasabah_id,omitempty" json:"nasabah_id,omitempty"`
	Action     string                 `bson:"action" json:"action"` // LOGIN, REGISTER, UPLOAD_KYC, UPDATE_PROFILE, LOGOUT
	IPAddress  string                 `bson:"ip_address" json:"ip_address"`
	UserAgent  string                 `bson:"user_agent" json:"user_agent"`
	Status     string                 `bson:"status" json:"status"` // success, failed
	Detail     map[string]interface{} `bson:"detail,omitempty" json:"detail,omitempty"`
	Timestamp  time.Time              `bson:"timestamp" json:"timestamp"`
}

// ========== REQUEST/RESPONSE DTOs ==========

type RegisterRequest struct {
	NIK          string  `json:"nik" binding:"required"`
	NamaLengkap  string  `json:"nama_lengkap" binding:"required"`
	Email        string  `json:"email" binding:"required,email"`
	Password     string  `json:"password" binding:"required,min=8"`
	NoHP         string  `json:"no_hp" binding:"required"`
	TanggalLahir string  `json:"tanggal_lahir" binding:"required"`
	Alamat       Address `json:"alamat" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string   `json:"token"`
	Nasabah   *Nasabah `json:"nasabah"`
	ExpiresIn int      `json:"expires_in"`
}

type UpdateProfileRequest struct {
	NoHP   string  `json:"no_hp,omitempty"`
	Alamat *Address `json:"alamat,omitempty"`
}

type UploadKYCRequest struct {
	TipeDokumen string `json:"tipe_dokumen" binding:"required"`
	NamaFile    string `json:"nama_file" binding:"required"`
	MimeType    string `json:"mime_type" binding:"required"`
	FileBase64  string `json:"file_base64" binding:"required"`
}
