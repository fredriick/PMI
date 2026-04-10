package gateway

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DeprecationService struct {
	SunsetDate  time.Time
	LegacyPaths []string
}

func NewDeprecationService() *DeprecationService {
	return &DeprecationService{
		SunsetDate: time.Now().Add(90 * 24 * time.Hour),
		LegacyPaths: []string{
			"/admin/nodes",
			"/admin/cooldowns",
			"/admin/sessions",
		},
	}
}

func (d *DeprecationService) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		for _, legacy := range d.LegacyPaths {
			if path == legacy || path == legacy+"/" {
				c.Header("Deprecation", `date="`+d.SunsetDate.Format(http.TimeFormat)+`", msg="Use /v1`+legacy+` instead"`)
				c.Header("Link", `</v1`+legacy+`>; rel="successor-version"`)
				c.Next()
				return
			}
		}

		c.Next()
	}
}

func (d *DeprecationService) GetSunsetDate() time.Time {
	return d.SunsetDate
}

func (d *DeprecationService) AddLegacyPath(path string) {
	d.LegacyPaths = append(d.LegacyPaths, path)
}
