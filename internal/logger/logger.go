package logger

import "go.uber.org/zap"

var (
	Plain *zap.Logger
	Sugar *zap.SugaredLogger
)

// Init builds both [Plain] and [Sugar] loggers
func Init() {
	Plain = zap.Must(zap.NewDevelopment())
	Sugar = Plain.Sugar()
	Plain.Info("logger initialized")
}

// Sync flushes buffered logs (if any). Call Sync before exiting application.
func Sync() {
	Plain.Info("logger synchronized")
	_ = Plain.Sync()
	_ = Sugar.Sync()
}
