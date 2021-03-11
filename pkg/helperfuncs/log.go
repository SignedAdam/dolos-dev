package helperfuncs

import (
	"fmt"
	"time"
)

func Log(format string, args ...interface{}) {
	fmt.Printf(fmt.Sprint(time.Now().Format("03:04:05 PM 01/02/06"), ": ", format, "\n"), args...)
}
