package ingest

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	tempDir        = "temp"
	cacheSize      = 100 * 1024 * 1024 // 100MB
	quotaSize      = 1 * 1024 * 1024 * 1024 // 1GB
	gcInterval     = 1 * time.Hour
	maxConcurrent  = 5
)

var (
	cache      = make(map[string][]byte)
	cacheMutex = &sync.RWMutex{}
)

func DownloadVideo(videoID string) (*os.File, error) {
	// Create a temporary directory to store the downloaded video
	tmpDir, err := os.MkdirTemp("", tempDir)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	// Check if the video is already in the cache
	cacheMutex.RLock()
	if cachedVideo, ok := cache[videoID]; ok {
		cacheMutex.RUnlock()
		return bytes.NewBuffer(cachedVideo), nil
	}
	cacheMutex.RUnlock()

	// Download the video in chunks
	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	resp, err := http.Get(videoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Create a new file to store the video
	videoFile, err := os.Create(filepath.Join(tmpDir, fmt.Sprintf("%s.mp4", videoID)))
	if err != nil {
		return nil, err
	}
	defer videoFile.Close()

	// Download the video in chunks
	buf := make([]byte, 1024*1024)
	for {
		n, err := resp.Body.Read(buf)
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}

		// Write the chunk to the file
		_, err = videoFile.Write(buf[:n])
		if err != nil {
			return nil, err
		}

		// Add the chunk to the cache
		cacheMutex.Lock()
		if len(cache[videoID]) > cacheSize {
			delete(cache, videoID)
		} else {
			cache[videoID] = append(cache[videoID], buf[:n]...)
		}
		cacheMutex.Unlock()
	}

	// Upload the video to S3
	bucketName := "your-bucket-name"
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-west-2")}, nil)
	if err != nil {
		return nil, err
	}
	s3Client := s3.New(sess)
	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(videoID),
		Body:   videoFile,
	})
	if err != nil {
		return nil, err
	}

	// Check if the quota has been exceeded
	quotaMutex.Lock()
	if quotaSize < getUsedDiskSpace() {
		log.Println("Quota exceeded!")
		quotaMutex.Unlock()
		return nil, fmt.Errorf("quota exceeded")
	}
	quotaMutex.Unlock()

	return videoFile, nil
}

func getUsedDiskSpace() int64 {
	// Get the current disk usage
	var stat syscall.Statfs_t
	err := syscall.Statfs(".", &stat)
	if err != nil {
		log.Println(err)
		return 0
	}
	return stat.Blocks * uint64(stat.Bsize)
}

func gc() {
	// Periodically clean up unused files and free up disk space
	ticker := time.NewTicker(gcInterval)
	defer ticker.Stop()
	for range ticker.C {
		// Remove unused files
		files, err := os.ReadDir(tempDir)
		if err != nil {
			log.Println(err)
			continue
		}
		for _, file := range files {
			if file.ModTime().Before(time.Now().Add(-gcInterval)) {
				err := os.Remove(filepath.Join(tempDir, file.Name()))
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
}

func main() {
	// Start the garbage collector
	go gc()

	// Download a video
	videoID := "some_video_id"
	videoFile, err := DownloadVideo(videoID)
	if err != nil {
		log.Fatal(err)
	}
	defer videoFile.Close()

	fmt.Println("Video downloaded successfully!")
}