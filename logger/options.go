package log

// Option for load profiles maybe
// eg. yml
// micro:
//   logger:
//     name:
//     dialect: zap/default/logrus
//     zap:
//       xxx:
//     logrus:
//       xxx:
type Option func(*Options)

type Options struct {
	Environment    string
	Dir            string
	FileMaxSize    int
	FileMaxBackups int
	FileMaxAge     int
	FileCompress   bool
	// more options
}
