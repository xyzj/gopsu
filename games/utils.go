package games

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
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
		c.Header("Content-Type", "text/html")
		c.Status(http.StatusOK)
		render.WriteString(c.Writer,
			`<h2>
			<a href="/game/2048">2048</a>
			</h2>
			<h2>
			<a href="/game/memory">memory</a>
			</h2>
			<h2>
			<a href="/game/music">music</a>
			</h2>
			<h2>
			<a href="/game/number">number</a>
			</h2>
			<h2>
			<a href="/game/snake">snake</a>
			</h2>
			<h2>
			<a href="/game/tetris">tetris</a>
			</h2>`,
			nil)
	}
}
