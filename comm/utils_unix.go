// +build !windows

package comm

import (
	"syscall"
)

func init() {
	// http://whaotingblog.blogspot.com/2017/12/golang-mkdir-permission-error.html
	syscall.Umask(0)
}
