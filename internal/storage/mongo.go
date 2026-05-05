package storage

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client
var ProductsCollection *mongo.Collection

func InitMongo(uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err = client.Ping(context.Background(), nil); err != nil {
		return nil, fmt.Errorf("mongo ping: %w", err)
	}
	MongoClient = client
	ProductsCollection = client.Database("shop").Collection("products")
	return client, nil
}
