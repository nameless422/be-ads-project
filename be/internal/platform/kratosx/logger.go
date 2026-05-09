package kratosx

import (
	"fmt"
	"log"
	"strings"

	klog "github.com/go-kratos/kratos/v2/log"
)

type stdLoggerAdapter struct {
	logger *log.Logger
}

func NewLogger(base *log.Logger, service string) klog.Logger {
	return klog.With(
		&stdLoggerAdapter{logger: base},
		"service", service,
		"ts", klog.DefaultTimestamp,
		"caller", klog.DefaultCaller,
	)
}

func (l *stdLoggerAdapter) Log(level klog.Level, keyvals ...any) error {
	fields := make([]string, 0, len(keyvals)/2+1)
	fields = append(fields, fmt.Sprintf("level=%s", level.String()))
	for i := 0; i < len(keyvals); i += 2 {
		key := fmt.Sprintf("key_%d", i)
		if i < len(keyvals) {
			key = fmt.Sprint(keyvals[i])
		}
		var value any
		if i+1 < len(keyvals) {
			value = keyvals[i+1]
		}
		fields = append(fields, fmt.Sprintf("%s=%v", key, value))
	}
	l.logger.Print(strings.Join(fields, " "))
	return nil
}
