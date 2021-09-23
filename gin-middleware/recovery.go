package ginmiddleware

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/xyzj/gopsu"
)

// var (
// 	dunno     = gopsu.Bytes("???")
// 	centerDot = gopsu.Bytes("·")
// 	dot       = gopsu.Bytes(".")
// 	slash     = gopsu.Bytes("/")
// )

// Recovery 错误恢复
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}
				if gin.DefaultWriter != nil {
					// stack := stack(3)
					httpRequest, _ := httputil.DumpRequest(c.Request, false)
					headers := strings.Split(string(httpRequest), "\r\n")
					for idx, header := range headers {
						current := strings.Split(header, ":")
						if current[0] == "Authorization" {
							headers[idx] = current[0] + ": *"
						}
					}
					if brokenPipe {
						fmt.Fprintf(gin.DefaultWriter, "%s\n%s\n", err, gopsu.String(httpRequest))
					} else if gin.IsDebugging() {
						fmt.Fprintf(gin.DefaultWriter, "[Recovery] %s\n%+v\n", strings.Join(headers, "\n"), errors.WithStack(err.(error)))
						// fmt.Fprintf(gin.DefaultWriter, "[Recovery] %s\n%+v\n", strings.Join(headers, "\n"), gopsu.String(stack))
					} else {
						fmt.Fprintf(gin.DefaultWriter, "[Recovery] %+v\n", errors.WithStack(err.(error)))
						// fmt.Fprintf(gin.DefaultWriter, "[Recovery] %+v\n", gopsu.String(stack))
					}
				}

				// If the connection is dead, we can't write a status to it.
				if brokenPipe {
					c.Error(err.(error)) // nolint: errcheck
					c.Abort()
				} else {
					c.Set("status", 0)
					c.Set("detail", "panic recovery, see log files for more information")
					c.Set("xfile", 6)
					c.AbortWithStatusJSON(http.StatusInternalServerError, c.Keys)
				}
			}
		}()
		c.Next()
	}
}

// // stack returns a nicely formatted stack frame, skipping skip frames.
// func stack(skip int) []byte {
// 	buf := new(bytes.Buffer) // the returned data
// 	// As we loop, we open files and read them. These variables record the currently
// 	// loaded file.
// 	var lines [][]byte
// 	var lastFile string
// 	for i := skip; ; i++ { // Skip the expected number of frames
// 		pc, file, line, ok := runtime.Caller(i)
// 		if !ok {
// 			break
// 		}
// 		// Print this much at least.  If we can't find the source, it won't show.
// 		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
// 		if file != lastFile {
// 			data, err := ioutil.ReadFile(file)
// 			if err != nil {
// 				continue
// 			}
// 			lines = bytes.Split(data, []byte{'\n'})
// 			lastFile = file
// 		}
// 		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
// 	}
// 	return buf.Bytes()
// }

// // source returns a space-trimmed slice of the n'th line.
// func source(lines [][]byte, n int) []byte {
// 	n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
// 	if n < 0 || n >= len(lines) {
// 		return dunno
// 	}
// 	return bytes.TrimSpace(lines[n])
// }

// // function returns, if possible, the name of the function containing the PC.
// func function(pc uintptr) []byte {
// 	fn := runtime.FuncForPC(pc)
// 	if fn == nil {
// 		return dunno
// 	}
// 	name := gopsu.Bytes(fn.Name())
// 	// The name includes the path name to the package, which is unnecessary
// 	// since the file name is already included.  Plus, it has center dots.
// 	// That is, we see
// 	//	runtime/debug.*T·ptrmethod
// 	// and want
// 	//	*T.ptrmethod
// 	// Also the package path might contains dot (e.g. code.google.com/...),
// 	// so first eliminate the path prefix
// 	if lastSlash := bytes.LastIndex(name, slash); lastSlash >= 0 {
// 		name = name[lastSlash+1:]
// 	}
// 	if period := bytes.Index(name, dot); period >= 0 {
// 		name = name[period+1:]
// 	}
// 	name = bytes.Replace(name, centerDot, dot, -1)
// 	return name

// }

// func timeFormat(t time.Time) string {
// 	var timeString = t.Format("2006/01/02 - 15:04:05")
// 	return timeString
// }
