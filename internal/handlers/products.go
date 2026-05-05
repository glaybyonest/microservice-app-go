package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go-microservice-app/internal/cache"
	"go-microservice-app/internal/models"
	"go-microservice-app/internal/storage"
)

const productsCacheTTL = 10 * time.Minute

func GetProducts(c *gin.Context) {
	ctx := c.Request.Context()
	cacheKey := "products:all"

	var products []models.Product
	if found, _ := cache.GetFromCache(cacheKey, &products); found {
		c.JSON(http.StatusOK, gin.H{"source": "cache", "data": products})
		return
	}

	cursor, err := storage.ProductsCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"_id": 1}))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &products); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if products == nil {
		products = []models.Product{}
	}

	_ = cache.SetToCache(cacheKey, products, productsCacheTTL)
	c.JSON(http.StatusOK, gin.H{"source": "server", "data": products})
}

func GetProduct(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")
	cacheKey := fmt.Sprintf("products:%s", id)

	var product models.Product
	if found, _ := cache.GetFromCache(cacheKey, &product); found {
		c.JSON(http.StatusOK, gin.H{"source": "cache", "data": product})
		return
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	err = storage.ProductsCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&product)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	_ = cache.SetToCache(cacheKey, product, productsCacheTTL)
	c.JSON(http.StatusOK, gin.H{"source": "server", "data": product})
}

func CreateProduct(c *gin.Context) {
	ctx := c.Request.Context()
	var input models.Product
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.ID = primitive.NewObjectID().Hex()
	input.CreatedAt = time.Now()
	input.UpdatedAt = time.Now()

	_, err := storage.ProductsCollection.InsertOne(ctx, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = cache.InvalidateCache("products:all")
	c.JSON(http.StatusCreated, gin.H{"data": input})
}

func UpdateProduct(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var input models.Product
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"name":        input.Name,
			"price":       input.Price,
			"category":    input.Category,
			"description": input.Description,
			"updated_at":  time.Now(),
		},
	}
	_, err = storage.ProductsCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = cache.InvalidateCache("products:all", fmt.Sprintf("products:%s", id))
	c.JSON(http.StatusOK, gin.H{"message": "Product updated"})
}

func DeleteProduct(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	_, err = storage.ProductsCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = cache.InvalidateCache("products:all", fmt.Sprintf("products:%s", id))
	c.JSON(http.StatusOK, gin.H{"message": "Product deleted"})
}
