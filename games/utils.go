package games

import (
	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/xyzj/gopsu/gin-middleware"
)

// GameGroup 游戏组
func GameGroup(c *gin.Context) {
	switch c.Param("game") {
	case "2048":
		Game2048(c)
	case "memory":
		GameMemory(c)
	case "music":
		GameMusic(c)
	case "number":
		GameNumber(c)
	case "snake":
		GameSnake(c)
	case "tetris":
		GameTetris(c)
	default:
		ginmiddleware.Page404(c)
	}
}
