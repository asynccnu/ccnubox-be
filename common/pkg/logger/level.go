package logger

import klog "github.com/go-kratos/kratos/v2/log"

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func toSelfLevel(l klog.Level) Level {
    switch l {
    case klog.LevelDebug:
        return DEBUG
    case klog.LevelInfo:
        return INFO
    case klog.LevelWarn:
        return WARN
    case klog.LevelError:
        return ERROR
    case klog.LevelFatal:
        return ERROR 
    default:
        return INFO 
    }
}