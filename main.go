package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	fmt.Println("=== PEAR S3-based CDN ===")

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "CDN OK"})
	})

	accessKey := os.Getenv("MinIOAccessKey")
	secretKey := os.Getenv("MinIOSecretKey")
	endpoint := os.Getenv("MinIOHost")
	region := os.Getenv("MinIORegion")
	useSSL := true

	minioOptions := &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: region,
	}

	minioClient, err := minio.New(endpoint, minioOptions)
	if err != nil {
		log.Fatalln(err)
	}

	r.GET("/public/:object", func(c *gin.Context) {
		bucketName := "cdn"
		objectName := c.Param("object")

		_, err = minioClient.StatObject(context.Background(), bucketName, objectName, minio.StatObjectOptions{})
		if err != nil {
			if minio.ToErrorResponse(err).Code == "NoSuchKey" {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "Object not found"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to get object"})
			return
		}

		object, err := minioClient.GetObject(context.Background(), bucketName, objectName, minio.GetObjectOptions{})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to get object"})
		}
		defer object.Close()

		c.DataFromReader(http.StatusOK, -1, "", object, nil)

	})

	r.Run(":3000")
}
