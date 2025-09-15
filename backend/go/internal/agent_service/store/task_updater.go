package store

import (
	"Jarvis_2.0/backend/go/internal/models"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

// TaskUpdater defines the interface for updating task records.
type TaskUpdater interface {
	UpdateTaskResult(ctx context.Context, taskID string, status models.TaskStatus, resultOrError interface{}) error
}

// MongoTaskUpdater is an implementation of TaskUpdater using MongoDB.
type MongoTaskUpdater struct {
	collection *mongo.Collection
}

// NewMongoTaskUpdater creates a new MongoTaskUpdater.
func NewMongoTaskUpdater(db *mongo.Database, collectionName string) *MongoTaskUpdater {
	return &MongoTaskUpdater{
		collection: db.Collection(collectionName),
	}
}

// UpdateTaskResult finds a task by its ID and updates its status, result/error, and completion time.
func (s *MongoTaskUpdater) UpdateTaskResult(ctx context.Context, taskID string, status models.TaskStatus, resultOrError interface{}) error {
	filter := bson.M{"_id": taskID}
	
	var update bson.M
	if status == models.TaskStatusSuccess {
		update = bson.M{
			"$set": bson.M{
				"status":       status,
				"result":       resultOrError,
				"completed_at": time.Now(),
			},
		}
	} else {
		// If status is failed, we expect resultOrError to be an error string.
		update = bson.M{
			"$set": bson.M{
				"status":       status,
				"error":        resultOrError,
				"completed_at": time.Now(),
			},
		}
	}

	_, err := s.collection.UpdateOne(ctx, filter, update)
	return err
}
