package gozero
     
import "os"   
import "runtime"
import vector "container/vector"

// #include "get_errno.c"
import "C"

// Datastructure for encapsulating locking a goroutine to an OSThread
// Do not use from multiple goroutines.  
//
// Lock may be released by calling Finish, this will execute all registered
// finalizers
type GoThread struct {
	locked	bool
	thunks	*vector.Vector
}

type OSThreadBound interface {
  OnOSThreadLock(thr *GoThread)
	OnOSThreadUnlock(thr *GoThread)
}

// Create a new GoThread and lock the current goroutine to the executing thread
func NewGoThread() *GoThread {
	var thr = new(GoThread)
	runtime.LockOSThread()
	thr.locked = true
	thr.thunks = new(vector.Vector)
	return thr
}

// Register a finalizer to be called when this GoThread finishes
func (p *GoThread) OnFinish(x OSThreadBound) {
	defer x.OnOSThreadLock(p)
	p.thunks.Push(x)
}

var AlreadyFinished = os.NewError("GoThread.Finish() called twice")

// Finish this GoThread and execute all finalizers that have been registered
// Panic otherwise
func (p *GoThread) Finish() {
	if (p.locked) { 
		defer runtime.UnlockOSThread()

		p.locked = false
		for i := 0; i < p.thunks.Len(); i++ {
			p.thunks.Pop().(OSThreadBound).OnOSThreadUnlock(p)
		}
	} else {
		panic(AlreadyFinished)
	}
}

func WithOSThread(fun func (thr *GoThread) interface{}) (interface{}) {
	var thr = NewGoThread()
	defer thr.Finish()

	return fun(thr)
}

// Deliver current errno from C.  
// For this to work reliably, you must lock the executing goroutine to the 
// underlying OSThread, i.e. by using GoThread!
func Errno() int { return int(C.get_errno()) }

type ErrnoFun func (int) os.Error

func CatchError(errnoFun ErrnoFun) {
  var c_errno = int(C.get_errno())
  if (c_errno != 0) {
    var error = errnoFun(c_errno)
    if (error == nil) {
      panic(os.NewSyscallError("Unexpected errno", c_errno))
    } else {
      panic(error)
    }
  }
}

// Panics with error if cond is true
func MayPanic(cond bool, error os.Error) {
	if (cond) { panic(error) }
}

