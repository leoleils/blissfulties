package main

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type FileInfo struct {
	Name    string    `json:"name"`
	ModTime time.Time `json:"mod_time"`
}

func main() {
	r := gin.Default()
	r.Static("/static", "./static")
	r.Static("/uploads", "./uploads")
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	r.GET("/gallery", func(c *gin.Context) {
		c.HTML(http.StatusOK, "gallery.html", nil)
	})

	r.GET("/upload", func(c *gin.Context) {
		c.HTML(http.StatusOK, "upload.html", nil)
	})

	r.POST("/upload", func(c *gin.Context) {
		form, _ := c.MultipartForm()
		files := form.File["files"]

		if len(files) > 20 {
			c.String(http.StatusBadRequest, "Too many files. Maximum is 20.")
			return
		}

		for _, file := range files {
			if file.Size > 1<<30 { // 1GB
				c.String(http.StatusBadRequest, "File too large. Maximum size is 1GB.")
				return
			}
			c.SaveUploadedFile(file, filepath.Join("uploads", file.Filename))
		}

		c.String(http.StatusOK, "Files uploaded successfully")
	})

	r.GET("/files", func(c *gin.Context) {
		files, err := os.ReadDir("uploads")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var fileInfos []FileInfo
		for _, file := range files {
			info, err := file.Info()
			if err != nil {
				continue
			}
			fileInfos = append(fileInfos, FileInfo{
				Name:    file.Name(),
				ModTime: info.ModTime(),
			})
		}

		sort.Slice(fileInfos, func(i, j int) bool {
			return fileInfos[i].ModTime.After(fileInfos[j].ModTime)
		})

		page, _ := c.GetQuery("page")
		pageSize := 9
		start := 0
		if page != "" {
			start = (atoi(page) - 1) * pageSize
		}
		if start > len(fileInfos) {
			start = len(fileInfos)
		}
		end := start + pageSize
		if end > len(fileInfos) {
			end = len(fileInfos)
		}

		c.JSON(http.StatusOK, gin.H{"files": fileInfos[start:end]})
	})

	r.Run(":8080")
}

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
