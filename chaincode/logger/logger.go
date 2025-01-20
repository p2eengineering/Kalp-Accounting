package logger

import (
	"github.com/muditp2e/kalp-sdk-public/kalpsdk"
)

var Log *kalpsdk.ChaincodeLogger

func init() {
	Log = kalpsdk.NewLogger()
}
