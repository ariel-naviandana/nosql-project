package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"banking-nosql/internal/models"
	"banking-nosql/internal/repository"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	mongoRepo *repository.MongoRepository
	redisRepo *repository.RedisRepository
	jwtSecret string
}

func NewAuthService(mongoRepo *repository.MongoRepository, redisRepo *repository.RedisRepository, jwtSecret string) *AuthService {
	return &AuthService{
		mongoRepo: mongoRepo,
		redisRepo: redisRepo,
		jwtSecret: jwtSecret,
	}
}

const SessionTTL = 24 * time.Hour
const RateLimitWindow = 15 * time.Minute
const MaxLoginAttempts = 5

func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.Nasabah, error) {
	// Cek apakah email sudah ada
	existing, _ := s.mongoRepo.FindNasabahByEmail(ctx, req.Email)
	if existing != nil {
		return nil, errors.New("email sudah terdaftar")
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	nasabah := &models.Nasabah{
		AccountID:    fmt.Sprintf("ACC-%s", uuid.New().String()[:8]),
		NIK:          req.NIK,
		NamaLengkap:  req.NamaLengkap,
		Email:        req.Email,
		Password:     string(hashed),
		NoHP:         req.NoHP,
		TanggalLahir: req.TanggalLahir,
		Alamat:       req.Alamat,
	}

	if err := s.mongoRepo.CreateNasabah(ctx, nasabah); err != nil {
		return nil, err
	}

	// Audit log
	s.mongoRepo.CreateAuditLog(ctx, &models.AuditLog{
		AccountID: nasabah.AccountID,
		NasabahID: nasabah.ID.Hex(),
		Action:    "REGISTER",
		Status:    "success",
		Detail: map[string]interface{}{
			"email": nasabah.Email,
			"nik":   nasabah.NIK,
		},
	})

	nasabah.Password = ""
	return nasabah, nil
}

func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest, ip, userAgent string) (*models.LoginResponse, error) {
	// Rate limit per IP
	attempts, err := s.redisRepo.IncrRateLimit(ctx, fmt.Sprintf("login:%s", ip), RateLimitWindow)
	if err != nil {
		return nil, err
	}
	if attempts > MaxLoginAttempts {
		// Audit log - rate limited
		s.mongoRepo.CreateAuditLog(ctx, &models.AuditLog{
			Action:    "LOGIN",
			IPAddress: ip,
			UserAgent: userAgent,
			Status:    "failed",
			Detail:    map[string]interface{}{"reason": "rate_limited", "email": req.Email},
		})
		return nil, errors.New("terlalu banyak percobaan login, coba lagi dalam 15 menit")
	}

	// Cari nasabah
	nasabah, err := s.mongoRepo.FindNasabahByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("email atau password salah")
	}

	// Verifikasi password
	if err := bcrypt.CompareHashAndPassword([]byte(nasabah.Password), []byte(req.Password)); err != nil {
		s.mongoRepo.CreateAuditLog(ctx, &models.AuditLog{
			AccountID: nasabah.AccountID,
			NasabahID: nasabah.ID.Hex(),
			Action:    "LOGIN",
			IPAddress: ip,
			UserAgent: userAgent,
			Status:    "failed",
			Detail:    map[string]interface{}{"reason": "wrong_password"},
		})
		return nil, errors.New("email atau password salah")
	}

	// Buat session token
	token := uuid.New().String()
	if err := s.redisRepo.SetSession(ctx, token, nasabah.ID.Hex(), SessionTTL); err != nil {
		return nil, err
	}

	// Audit log - success
	s.mongoRepo.CreateAuditLog(ctx, &models.AuditLog{
		AccountID: nasabah.AccountID,
		NasabahID: nasabah.ID.Hex(),
		Action:    "LOGIN",
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    "success",
	})

	nasabah.Password = ""
	return &models.LoginResponse{
		Token:     token,
		Nasabah:   nasabah,
		ExpiresIn: int(SessionTTL.Seconds()),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, token string, nasabah *models.Nasabah, ip, userAgent string) error {
	// Hapus session
	if err := s.redisRepo.DeleteSession(ctx, token); err != nil {
		return err
	}
	// Blacklist token
	s.redisRepo.BlacklistToken(ctx, token, SessionTTL)

	// Audit log
	s.mongoRepo.CreateAuditLog(ctx, &models.AuditLog{
		AccountID: nasabah.AccountID,
		NasabahID: nasabah.ID.Hex(),
		Action:    "LOGOUT",
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    "success",
	})
	return nil
}

func (s *AuthService) ValidateToken(ctx context.Context, token string) (string, error) {
	// Cek blacklist
	blacklisted, err := s.redisRepo.IsTokenBlacklisted(ctx, token)
	if err != nil {
		return "", err
	}
	if blacklisted {
		return "", errors.New("token sudah tidak valid")
	}

	nasabahID, err := s.redisRepo.GetSession(ctx, token)
	if err != nil {
		return "", errors.New("session tidak ditemukan atau sudah expired")
	}
	return nasabahID, nil
}
