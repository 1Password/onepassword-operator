package logs

// A Level is a logging priority. Lower levels are more important.
// All levels have been multipled by -1 to ensure compatibilty
// between zapcore and logr
type Level int

const (
	ErrorLevel Level = iota - 2
	WarnLevel
	InfoLevel
	DebugLevel
)
