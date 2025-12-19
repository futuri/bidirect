package logging

import (
	"log"
	"os"
)

var logger = log.New(os.Stderr, "", log.LstdFlags)

func Infof(format string, v ...any) {
	logger.Printf("[INFO] "+format, v...)
}

func Errorf(format string, v ...any) {
	logger.Printf("[ERROR] "+format, v...)
}

func Fatalf(format string, v ...any) {
	logger.Fatalf("[FATAL] "+format, v...)
}
