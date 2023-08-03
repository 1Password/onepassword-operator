package logs

// A Level is a logging priority. Lower levels are more important.
// All levels have been multipled by -1 to ensure compatibilty
// between zapcore and logr
const (
	ErrorLevel = -2
	WarnLevel  = -1
	InfoLevel  = 0
	DebugLevel = 1
)
