package repository

import (
	"context"
	"time"

	"banking-nosql/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepository struct {
	db *mongo.Database
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{db: db}
}

// ========== NASABAH ==========

func (r *MongoRepository) CreateNasabah(ctx context.Context, nasabah *models.Nasabah) error {
	nasabah.ID = primitive.NewObjectID()
	nasabah.CreatedAt = time.Now()
	nasabah.UpdatedAt = time.Now()
	nasabah.StatusKYC = "pending"

	_, err := r.db.Collection("nasabah").InsertOne(ctx, nasabah)
	return err
}

func (r *MongoRepository) FindNasabahByEmail(ctx context.Context, email string) (*models.Nasabah, error) {
	var nasabah models.Nasabah
	err := r.db.Collection("nasabah").FindOne(ctx, bson.M{"email": email}).Decode(&nasabah)
	if err != nil {
		return nil, err
	}
	return &nasabah, nil
}

func (r *MongoRepository) FindNasabahByID(ctx context.Context, id primitive.ObjectID) (*models.Nasabah, error) {
	var nasabah models.Nasabah
	err := r.db.Collection("nasabah").FindOne(ctx, bson.M{"_id": id}).Decode(&nasabah)
	if err != nil {
		return nil, err
	}
	return &nasabah, nil
}

func (r *MongoRepository) UpdateNasabah(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	update["updated_at"] = time.Now()
	_, err := r.db.Collection("nasabah").UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": update},
	)
	return err
}

func (r *MongoRepository) FindAllNasabah(ctx context.Context, page, limit int64) ([]models.Nasabah, int64, error) {
	skip := (page - 1) * limit
	opts := options.Find().SetSkip(skip).SetLimit(limit).SetProjection(bson.M{"password": 0})

	cursor, err := r.db.Collection("nasabah").Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var results []models.Nasabah
	if err := cursor.All(ctx, &results); err != nil {
		return nil, 0, err
	}

	total, err := r.db.Collection("nasabah").CountDocuments(ctx, bson.M{})
	return results, total, err
}

// ========== KYC ==========

func (r *MongoRepository) CreateDokumenKYC(ctx context.Context, doc *models.DokumenKYC) error {
	doc.ID = primitive.NewObjectID()
	doc.UploadedAt = time.Now()
	doc.Status = "uploaded"

	_, err := r.db.Collection("dokumen_kyc").InsertOne(ctx, doc)
	return err
}

func (r *MongoRepository) FindKYCByNasabah(ctx context.Context, nasabahID primitive.ObjectID) ([]models.DokumenKYC, error) {
	cursor, err := r.db.Collection("dokumen_kyc").Find(ctx, bson.M{"nasabah_id": nasabahID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []models.DokumenKYC
	err = cursor.All(ctx, &docs)
	return docs, err
}

func (r *MongoRepository) UpdateKYCStatus(ctx context.Context, id primitive.ObjectID, status, keterangan string) error {
	now := time.Now()
	update := bson.M{
		"status":      status,
		"keterangan":  keterangan,
		"verified_at": now,
	}
	_, err := r.db.Collection("dokumen_kyc").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	return err
}

// ========== AUDIT LOG ==========

func (r *MongoRepository) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	log.ID = primitive.NewObjectID()
	log.Timestamp = time.Now()

	_, err := r.db.Collection("audit_log").InsertOne(ctx, log)
	return err
}

func (r *MongoRepository) FindAuditLogByAccount(ctx context.Context, accountID string, page, limit int64) ([]models.AuditLog, int64, error) {
	skip := (page - 1) * limit
	opts := options.Find().SetSkip(skip).SetLimit(limit).SetSort(bson.M{"timestamp": -1})

	filter := bson.M{}
	if accountID != "" {
		filter["account_id"] = accountID
	}

	cursor, err := r.db.Collection("audit_log").Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var logs []models.AuditLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, 0, err
	}

	total, _ := r.db.Collection("audit_log").CountDocuments(ctx, filter)
	return logs, total, nil
}

// ========== INDEX SETUP ==========

func (r *MongoRepository) SetupIndexes(ctx context.Context) error {
	// Nasabah indexes
	r.db.Collection("nasabah").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"email": 1},
		Options: options.Index().SetUnique(true),
	})
	r.db.Collection("nasabah").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"nik": 1},
		Options: options.Index().SetUnique(true),
	})
	r.db.Collection("nasabah").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"account_id": 1},
	})

	// KYC indexes
	r.db.Collection("dokumen_kyc").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"nasabah_id": 1},
	})

	// Audit log indexes
	r.db.Collection("audit_log").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"account_id": 1},
	})
	r.db.Collection("audit_log").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"timestamp": -1},
	})

	return nil
}
