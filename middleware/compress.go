package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
)

// Compress returns a middleware that compresses HTTP responses using gzip, deflate, or brotli
// based on the Accept-Encoding header.
//
// Compression levels:
// - 1: Best speed (fastest, lower compression ratio)
// - 2: Balanced (default - good speed and compression)
// - 3: Best compression (slowest, highest compression ratio)
func Compress(level int) fiber.Handler {
	// Map our 1-3 scale to Fiber's compress.Level constants
	var compressionLevel compress.Level
	switch level {
	case 1:
		compressionLevel = compress.LevelBestSpeed
	case 2:
		compressionLevel = compress.LevelDefault
	case 3:
		compressionLevel = compress.LevelBestCompression
	default:
		compressionLevel = compress.LevelDefault
	}

	return compress.New(compress.Config{
		Level: compressionLevel,
	})
}
