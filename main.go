package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

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

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		log.Fatalln("Failed to initialize MinIO client:", err)
	}

	r.GET("/public/:object", func(c *gin.Context) {
		bucketName := "cdn"
		objectName := c.Param("object")

		stat, err := minioClient.StatObject(context.Background(), bucketName, objectName, minio.StatObjectOptions{})
		if err != nil {
			errResp := minio.ToErrorResponse(err)
			if errResp.Code == "NoSuchKey" || errResp.Code == "NoSuchObject" {
				c.Status(http.StatusNotFound)
				c.Writer.Write([]byte("404 Not Found"))
				return
			}
			c.Status(http.StatusInternalServerError)
			c.Writer.Write([]byte("Internal Server Error"))
			return
		}

		object, err := minioClient.GetObject(context.Background(), bucketName, objectName, minio.GetObjectOptions{})
		if err != nil {
			c.Status(http.StatusInternalServerError)
			c.Writer.Write([]byte("Failed to get object"))
			return
		}
		defer object.Close()

		contentType := stat.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		c.Header("Content-Type", contentType)
		c.Header("Content-Length", strconv.FormatInt(stat.Size, 10))
		c.Header("Accept-Ranges", "bytes")

		c.DataFromReader(http.StatusOK, stat.Size, contentType, object, nil)
	})

	r.Run(":3000")
}
