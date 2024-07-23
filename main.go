package main

import (
    "net/http"
    "os"
    "path/filepath"

    "github.com/gin-gonic/gin"
)

const (
    MAX_FILES = 30
    MAX_SIZE  = 2 * 1024 * 1024 * 1024 // 2GB
)

func main() {
    router := gin.Default()
    router.Static("/uploads", "./uploads")
    router.Static("/static", "./static")
    router.LoadHTMLGlob("templates/*")

    router.GET("/", func(c *gin.Context) {
        c.HTML(http.StatusOK, "index.html", nil)
    })

    router.GET("/upload", func(c *gin.Context) {
        c.HTML(http.StatusOK, "upload.html", nil)
    })

    router.POST("/upload", func(c *gin.Context) {
        form, err := c.MultipartForm()
        if err != nil {
            c.String(http.StatusBadRequest, "Bad request")
            return
        }

        files := form.File["files"]
        if len(files) > MAX_FILES {
            c.String(http.StatusBadRequest, "You can upload a maximum of 30 images at a time.")
            return
        }

        for _, file := range files {
            if file.Size > MAX_SIZE {
                c.String(http.StatusBadRequest, "Each file must be less than 2GB.")
                return
            }

            if err := c.SaveUploadedFile(file, filepath.Join("uploads", file.Filename)); err != nil {
                c.String(http.StatusInternalServerError, "Could not save file")
                return
            }
        }

        c.String(http.StatusOK, "Files uploaded successfully")
    })

    router.GET("/files", func(c *gin.Context) {
        files, err := os.ReadDir("uploads")
        if err != nil {
            c.String(http.StatusInternalServerError, "Could not read directory")
            return
        }

        var fileNames []string
        for _, file := range files {
            fileNames = append(fileNames, file.Name())
        }

        c.JSON(http.StatusOK, gin.H{"files": fileNames})
    })

    router.Run(":8080")
}