package log

type logStream struct {
	stream <-chan Record
	stop   chan bool
}

func (l *logStream) Chan() <-chan Record {
	return l.stream
}

func (l *logStream) Stop() error {
	select {
	case <-l.stop:
		return nil
	default:
		close(l.stop)
	}
	return nil
}
