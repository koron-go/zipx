package zipx

// Progress holds progress of extraction.
type Progress struct {
	// NumDone is number of extracted files.
	NumDone int

	// NumTotal is number of total files.
	NumTotal int
}

// Monitor monitors progress of extraction.
type Monitor interface {
	Monitor(Progress)
}

// MonitorFunc is used to implement Monitor by function.
type MonitorFunc func(Progress)

// Monitor monitors progress of extraction.
func (f MonitorFunc) Monitor(p Progress) {
	f(p)
}

// NullMonitor is monitor which ignores progress entirely.
var NullMonitor = MonitorFunc(func(_ Progress) {
	// nothing to do.
})
