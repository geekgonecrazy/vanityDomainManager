package router

import (
	"github.com/geekgonecrazy/vanityDomainManager/jobs"
	"github.com/geekgonecrazy/vanityDomainManager/queueManager"
	"github.com/gin-gonic/gin"
)

func Start() {
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.POST("/v1/jobs", func(c *gin.Context) {
		var job jobs.VanityDomainJob
		if err := c.BindJSON(&job); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		if err := queueManager.Mgr().AddDomainJob(job); err != nil {
			c.JSON(500, gin.H{"error": "Failed to add job to queue"})
			return
		}
	})

	router.Run(":9595")
}
