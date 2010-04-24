package zmq
     
import "os"   
import "runtime"

// #include "get_errno.c"
import "C"



// ******** Closeable ********

// Interface for all artefacts that initially are open and later 
// may be closed/terminated
type Closeable interface { Close() }



// ******** Thunks ********

type Thunk func()

// Helper for calling thunk within a separate go routine bound to a fixed OSThread
func (p Thunk) GoOSThread() {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		p()
	}()
}

// Reference counter
type RefC uint32

// Create new reference counter that will call thunk when done
// (Instantly spawns a goroutine)
func (p Thunk) NewRefC(initialCount uint32) *RefC { 
	ref := new(RefC)
	*ref = RefC(initialCount)
	go func() { defer p(); ref.Decr(); }()
	return ref
}

func (p *RefC) Incr() { runtime.Semrelease((*uint32)(p)) } 
func (p *RefC) Decr() { runtime.Semacquire((*uint32)(p)) } 




// ******** Error Handling ********

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

