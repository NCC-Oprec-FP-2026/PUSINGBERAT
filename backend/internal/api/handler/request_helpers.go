package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func boolQuery(c *gin.Context, key string) *bool {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil
	}
	return &value
}

func timeQuery(c *gin.Context, key string) *time.Time {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}

	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	return &value
}
