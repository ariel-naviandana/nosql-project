package handlers

import (
	"net/http"
	"strconv"

	"banking-nosql/internal/middleware"
	"banking-nosql/internal/models"
	"banking-nosql/internal/repository"
	"banking-nosql/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Handler struct {
	mongoRepo *repository.MongoRepository
	redisRepo *repository.RedisRepository
	authSvc   *services.AuthService
}

func New(mongoRepo *repository.MongoRepository, redisRepo *repository.RedisRepository, authSvc *services.AuthService) *Handler {
	return &Handler{
		mongoRepo: mongoRepo,
		redisRepo: redisRepo,
		authSvc:   authSvc,
	}
}

// ========== AUTH ==========

func (h *Handler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nasabah, err := h.authSvc.Register(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "registrasi berhasil",
		"data":    nasabah,
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authSvc.Login(c.Request.Context(), &req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "login berhasil",
		"data":    resp,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	nasabah := middleware.GetNasabah(c)
	token := middleware.GetToken(c)

	if err := h.authSvc.Logout(c.Request.Context(), token, nasabah, c.ClientIP(), c.Request.UserAgent()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logout berhasil"})
}

// ========== PROFIL NASABAH ==========

func (h *Handler) GetProfile(c *gin.Context) {
	nasabah := middleware.GetNasabah(c)
	c.JSON(http.StatusOK, gin.H{"data": nasabah})
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	nasabah := middleware.GetNasabah(c)
	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := bson.M{}
	if req.NoHP != "" {
		update["no_hp"] = req.NoHP
	}
	if req.Alamat != nil {
		update["alamat"] = req.Alamat
	}

	if err := h.mongoRepo.UpdateNasabah(c.Request.Context(), nasabah.ID, update); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.mongoRepo.CreateAuditLog(c.Request.Context(), &models.AuditLog{
		AccountID: nasabah.AccountID,
		NasabahID: nasabah.ID.Hex(),
		Action:    "UPDATE_PROFILE",
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    "success",
		Detail:    map[string]interface{}{"fields_updated": update},
	})

	c.JSON(http.StatusOK, gin.H{"message": "profil berhasil diperbarui"})
}

func (h *Handler) GetAllNasabah(c *gin.Context) {
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	nasabahList, total, err := h.mongoRepo.FindAllNasabah(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  nasabahList,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// ========== KYC ==========

func (h *Handler) UploadKYC(c *gin.Context) {
	nasabah := middleware.GetNasabah(c)
	var req models.UploadKYCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc := &models.DokumenKYC{
		NasabahID:   nasabah.ID,
		AccountID:   nasabah.AccountID,
		TipeDokumen: req.TipeDokumen,
		NamaFile:    req.NamaFile,
		MimeType:    req.MimeType,
		FileBase64:  req.FileBase64,
	}

	if err := h.mongoRepo.CreateDokumenKYC(c.Request.Context(), doc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.mongoRepo.CreateAuditLog(c.Request.Context(), &models.AuditLog{
		AccountID: nasabah.AccountID,
		NasabahID: nasabah.ID.Hex(),
		Action:    "UPLOAD_KYC",
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    "success",
		Detail: map[string]interface{}{
			"tipe_dokumen": req.TipeDokumen,
			"nama_file":    req.NamaFile,
		},
	})

	// Update status KYC nasabah jika dokumen pertama
	h.mongoRepo.UpdateNasabah(c.Request.Context(), nasabah.ID, bson.M{"status_kyc": "in_review"})

	doc.FileBase64 = "" // jangan return base64
	c.JSON(http.StatusCreated, gin.H{
		"message": "dokumen KYC berhasil diupload",
		"data":    doc,
	})
}

func (h *Handler) GetMyKYC(c *gin.Context) {
	nasabah := middleware.GetNasabah(c)
	docs, err := h.mongoRepo.FindKYCByNasabah(c.Request.Context(), nasabah.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Hapus base64 dari response list
	for i := range docs {
		docs[i].FileBase64 = ""
	}

	c.JSON(http.StatusOK, gin.H{"data": docs})
}

func (h *Handler) VerifyKYC(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var body struct {
		Status     string `json:"status" binding:"required"`
		Keterangan string `json:"keterangan"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.mongoRepo.UpdateKYCStatus(c.Request.Context(), objID, body.Status, body.Keterangan); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status KYC berhasil diperbarui"})
}

// ========== AUDIT LOG ==========

func (h *Handler) GetAuditLog(c *gin.Context) {
	nasabah := middleware.GetNasabah(c)
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "20"), 10, 64)

	logs, total, err := h.mongoRepo.FindAuditLogByAccount(c.Request.Context(), nasabah.AccountID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handler) GetAllAuditLog(c *gin.Context) {
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "20"), 10, 64)

	logs, total, err := h.mongoRepo.FindAuditLogByAccount(c.Request.Context(), "", page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// ========== REDIS INFO (untuk demo) ==========

func (h *Handler) GetSessionInfo(c *gin.Context) {
	token := middleware.GetToken(c)
	nasabah := middleware.GetNasabah(c)

	ttl, _ := h.redisRepo.GetSessionTTL(c.Request.Context(), token)
	rateLimitCount, _ := h.redisRepo.GetRateLimit(c.Request.Context(), "login:"+c.ClientIP())
	rateLimitTTL, _ := h.redisRepo.GetRateLimitTTL(c.Request.Context(), "login:"+c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"session": gin.H{
			"token_preview":  token[:8] + "...",
			"nasabah_id":     nasabah.ID.Hex(),
			"ttl_seconds":    int(ttl.Seconds()),
			"redis_key":      "session:" + token[:8] + "...",
		},
		"rate_limit": gin.H{
			"ip":           c.ClientIP(),
			"login_attempts": rateLimitCount,
			"reset_in_seconds": int(rateLimitTTL.Seconds()),
			"max_attempts": services.MaxLoginAttempts,
		},
	})
}
