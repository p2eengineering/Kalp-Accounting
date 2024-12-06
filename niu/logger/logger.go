package logger

import (
	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

var Log *kalpsdk.ChaincodeLogger

func init() {
	// Initialize the logger
	Log = kalpsdk.NewLogger()
}
