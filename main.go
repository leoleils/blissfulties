package main

import (
	"fmt"
	"image"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/rwcarlsen/goexif/exif"
)

type FileInfo struct {
	Name      string    `json:"name"`
	ModTime   time.Time `json:"mod_time"`
	Thumbnail string    `json:"thumbnail"`
}

func main() {
	r := gin.Default()
	r.Static("/static", "./static")
	r.Static("/uploads", "./uploads")
	r.Static("/thumbnails", "./thumbnails")
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
			filePath := filepath.Join("uploads", file.Filename)
			c.SaveUploadedFile(file, filePath)

			// Open the original image
			img, err := imaging.Open(filePath)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to open image: %s", err.Error())
				return
			}

			// Correct the image orientation based on EXIF data
			img = correctOrientation(filePath, img)

			// Generate thumbnail from the resized image
			thumbnail := imaging.Thumbnail(img, img.Bounds().Dx(), img.Bounds().Dy(), imaging.Box)
			thumbnailPath := filepath.Join("thumbnails", file.Filename)
			err = imaging.Save(thumbnail, thumbnailPath)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to save thumbnail: %s", err.Error())
				return
			}
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

			thumbnailPath := filepath.Join("thumbnails", file.Name())
			if _, err := os.Stat(thumbnailPath); os.IsNotExist(err) {
				// Generate thumbnail if it doesn't exist
				img, err := imaging.Open(filepath.Join("uploads", file.Name()))
				if err != nil {
					continue
				}

				// Correct the image orientation based on EXIF data
				img = correctOrientation(filepath.Join("uploads", file.Name()), img)
				// Generate thumbnail from the resized image
				thumbnail := imaging.Thumbnail(img, img.Bounds().Dx(), img.Bounds().Dy(), imaging.Box)
				err = imaging.Save(thumbnail, thumbnailPath, imaging.JPEGQuality(20))
				if err != nil {
					continue
				}
			}

			fileInfos = append(fileInfos, FileInfo{
				Name:      file.Name(),
				ModTime:   info.ModTime(),
				Thumbnail: "/thumbnails/" + file.Name(),
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

	r.GET("/download/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		filepath := filepath.Join("uploads", filename)
		c.File(filepath)
	})

	r.Run(":8080")
}

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func correctOrientation(filePath string, img image.Image) image.Image {
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Failed to open file for EXIF reading: %s\n", err.Error())
		return img
	}
	defer f.Close()

	exifData, err := exif.Decode(f)
	if err != nil {
		fmt.Printf("Failed to decode EXIF data: %s\n", err.Error())
		return img
	}

	orientation, err := exifData.Get(exif.Orientation)
	if err != nil {
		fmt.Printf("Failed to get EXIF orientation: %s\n", err.Error())
		return img
	}

	orientationValue, err := orientation.Int(0)
	if err != nil {
		fmt.Printf("Failed to parse EXIF orientation value: %s\n", err.Error())
		return img
	}

	switch orientationValue {
	case 2:
		img = imaging.FlipH(img)
	case 3:
		img = imaging.Rotate180(img)
	case 4:
		img = imaging.FlipV(img)
	case 5:
		img = imaging.Transpose(img)
	case 6:
		img = imaging.Rotate270(img)
	case 7:
		img = imaging.Transverse(img)
	case 8:
		img = imaging.Rotate90(img)
	}

	return img
}
