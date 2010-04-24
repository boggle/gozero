package zmq
     
import "os"   
import "runtime"
import vector "container/vector"

// #include "get_errno.c"
import "C"



// ******** Closeable ********

// (Thread-safe) interface for all artefacts that initially are open and later 
// may be closed/terminated
type Closeable interface {
	CondDo(openState bool, thunk func()) bool
	Close()
}

var errAlreadyClosed = os.NewError("Closeable already closed yet expected to be open")

// Thrown if a Closeable already has been closed (Terminated, Finished, ...)
func ErrAlreadyClosed() os.Error { return errAlreadyClosed }



// ******** GoThreads ********

// Datastructure for encapsulating locking a goroutine to an OSThread
// Do not use from multiple goroutines.  
//
// Lock may be released by calling Finish, this will execute all registered
// finalizers
type GoThread struct {
	Closeable

	locked	bool							// true if GoThread has locked on OSThread
	thunks	*vector.Vector		// Vector of thunks to be called on UnlockOSThread
}

// Type of all artefacts that may be tied to a GoThread
type OSThreadBound interface {
  OnOSThreadLock(thr *GoThread)
	OnOSThreadUnlock(thr *GoThread)
}

// Create a new GoThread and lock the current goroutine to the 
// executing OSThread
func NewGoThread() *GoThread {
	thr := new(GoThread)
	runtime.LockOSThread()
	thr.locked = true
	thr.thunks = new(vector.Vector)
	return thr
}

// Calls thunk if this GoThread is open and openState is true or
// Calls thunk if this GoThread is closed and openState is false.
// Returns true iff thunk was called, false otherwise.
func (p *GoThread) CondDo(openState bool, thunk func()) bool {
	if (p.locked == openState) { thunk(); return true; }
	return false;
}

// Panics unless this GoThread is still locked on its OSThread
func (p *GoThread) EnsureOSThreadBound() {
	if (! p.locked) { panic(errAlreadyClosed) }
}

// Register a finalizer to be called when this GoThread finishes
func (p *GoThread) Register(x OSThreadBound) {
	p.EnsureOSThreadBound()
	defer x.OnOSThreadLock(p)

	p.thunks.Push(x)
}


// Finish this GoThread and execute all finalizers that have been registered
// Panic otherwise
func (p *GoThread) Close() {
	if (p.locked) { 
		defer p.unlock()

		for i := 0; i < p.thunks.Len(); i++ {
			p.thunks.Pop().(OSThreadBound).OnOSThreadUnlock(p)
		}
	} else {
		panic(errAlreadyClosed)
	}
}

func (p *GoThread) unlock() {
		p.locked = false
		runtime.UnlockOSThread()
}

// Functions running in an independent OSThreadBound go routine
type OSTFun func (*GoThread, chan interface{})

// Helper for calling thunk within a separate go routine bound to an OSThread
func WithOSThread(thunk OSTFun) (chan interface{}) {
	ch := make(chan interface{})

	go func() {
		thr := NewGoThread()
		defer thr.Close()

		thunk(thr, ch)
	}()
	return ch
}



// ******** ERROR HANDLING ********

// Panics with error if cond is true
func CondPanic(cond bool, error os.Error) {
	if (cond) { panic(error) }
}

// Deliver current errno from C.  
// For this to work reliably, you must lock the executing goroutine to the 
// underlying OSThread, i.e. by using GoThread!
func errno() os.Errno { return os.Errno(uint64(C.get_errno())) }

// Type of Errno() to os.Error conversion functions
type ErrnoFun func (os.Errno) os.Error

// Calls CatchError(errnoFun) iff cond is true.
// Requires that the executing go routine has been locked to an OSThread.
func CondCatchError(cond bool, errnoFun ErrnoFun) {
	if (cond) { CatchError(errnoFun) }
}

// Gets errno from C and converts it into an os.Error using errnoFun.
// Requires that the executing go routine has been locked to an OSThread.
func CatchError(errnoFun ErrnoFun) {
  c_errno := errno()
  if (c_errno != os.Errno(0)) {
    error := errnoFun(c_errno)
    if (error == nil) {
      panic(os.Error(c_errno))
    } else {
      panic(error)
    }
  }
}

// {}

