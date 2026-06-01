package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Config holds MongoDB connection configuration
type Config struct {
	URI      string
	Database string
}

// Client wraps the MongoDB client for simplified usage
type Client struct {
	client *mongo.Client
	db     *mongo.Database
}

// NewClient creates a new MongoDB client and establishes connection
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.URI == "" {
		return nil, fmt.Errorf("mongodb URI cannot be empty")
	}
	if cfg.Database == "" {
		return nil, fmt.Errorf("mongodb database name cannot be empty")
	}

	clientOpts := options.Client().ApplyURI(cfg.URI)
	mongoClient, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	// Ping to verify connection
	if err := mongoClient.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return &Client{
		client: mongoClient,
		db:     mongoClient.Database(cfg.Database),
	}, nil
}

// Disconnect closes the MongoDB connection
func (c *Client) Disconnect(ctx context.Context) error {
	if c.client == nil {
		return nil
	}
	return c.client.Disconnect(ctx)
}

// GetDatabase returns the database instance
func (c *Client) GetDatabase() *mongo.Database {
	return c.db
}

// GetCollection returns a collection from the database
func (c *Client) GetCollection(collectionName string) *mongo.Collection {
	return c.db.Collection(collectionName)
}

// InsertOne inserts a single document into the specified collection
func (c *Client) InsertOne(ctx context.Context, collectionName string, document interface{}) (interface{}, error) {
	collection := c.GetCollection(collectionName)
	result, err := collection.InsertOne(ctx, document)
	if err != nil {
		return nil, fmt.Errorf("failed to insert document: %w", err)
	}
	return result.InsertedID, nil
}

// FindOne retrieves a single document from the specified collection
func (c *Client) FindOne(ctx context.Context, collectionName string, filter interface{}) *mongo.SingleResult {
	collection := c.GetCollection(collectionName)
	return collection.FindOne(ctx, filter)
}

// UpdateOne updates a single document in the specified collection
func (c *Client) UpdateOne(ctx context.Context, collectionName string, filter, update interface{}) (int64, error) {
	collection := c.GetCollection(collectionName)
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to update document: %w", err)
	}
	return result.ModifiedCount, nil
}

// DeleteOne deletes a single document from the specified collection
func (c *Client) DeleteOne(ctx context.Context, collectionName string, filter interface{}) (int64, error) {
	collection := c.GetCollection(collectionName)
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete document: %w", err)
	}
	return result.DeletedCount, nil
}

// IsConnected checks if the MongoDB client is connected
func (c *Client) IsConnected(ctx context.Context) bool {
	if c.client == nil {
		return false
	}
	return c.client.Ping(ctx, nil) == nil
}
