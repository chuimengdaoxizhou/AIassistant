package store

import (
	"Jarvis_2.0/backend/go/internal/models"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TaskStore defines the interface for task persistence.
type TaskStore interface {
	Create(ctx context.Context, task *models.TaskRecord) error
	GetByID(ctx context.Context, id string) (*models.TaskRecord, error)
	GetByUserID(ctx context.Context, userID string, page, limit int) ([]*models.TaskRecord, error)
	Update(ctx context.Context, task *models.TaskRecord) error
}

// MongoTaskStore is an implementation of TaskStore using MongoDB.
type MongoTaskStore struct {
	collection *mongo.Collection
}

// NewMongoTaskStore creates a new MongoTaskStore.
func NewMongoTaskStore(db *mongo.Database, collectionName string) *MongoTaskStore {
	return &MongoTaskStore{
		collection: db.Collection(collectionName),
	}
}

// Create inserts a new task record into the database.
func (s *MongoTaskStore) Create(ctx context.Context, task *models.TaskRecord) error {
	_, err := s.collection.InsertOne(ctx, task)
	return err
}

// GetByID retrieves a task by its ID.
func (s *MongoTaskStore) GetByID(ctx context.Context, id string) (*models.TaskRecord, error) {
	var task models.TaskRecord
	err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Or a custom not found error
		}
		return nil, err
	}
	return &task, nil
}

// GetByUserID retrieves a paginated list of tasks for a specific user.
func (s *MongoTaskStore) GetByUserID(ctx context.Context, userID string, page, limit int) ([]*models.TaskRecord, error) {
	var tasks []*models.TaskRecord
	opts := options.Find()
	opts.SetSort(bson.D{{"submitted_at", -1}}) // Sort by submission date descending
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))

	cursor, err := s.collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// Update updates an existing task record.
func (s *MongoTaskStore) Update(ctx context.Context, task *models.TaskRecord) error {
	filter := bson.M{"_id": task.ID}
	update := bson.M{
		"$set": bson.M{
			"status":       task.Status,
			"result":       task.Result,
			"error":        task.Error,
			"completed_at": task.CompletedAt,
		},
	}
	_, err := s.collection.UpdateOne(ctx, filter, update)
	return err
}
