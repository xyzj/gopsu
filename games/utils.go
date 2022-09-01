/*
Package games : 搜罗的一些网页小游戏

try /xgames/ in every go http server build by wlstmicro/v2
*/
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
	case "list":
		c.Header("Content-Type", "text/html")
		c.Status(http.StatusOK)
		render.WriteString(c.Writer,
			`<h2>
			<a href="./2048">2048</a>
			</h2>
			<h2>
			<a href="./memory">memory</a>
			</h2>
			<h2>
			<a href="./music">music</a>
			</h2>
			<h2>
			<a href="./number">number</a>
			</h2>
			<h2>
			<a href="./snake">snake</a>
			</h2>
			<h2>
			<a href="./tetris">tetris</a>
			</h2>`,
			nil)
	default:
		c.Abort()
	}
}
