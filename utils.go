package zmq

import "os"
import rt "runtime"
import "strconv"

// #include "get_errno.c"
import "C"


// ******** Thunks ********

// A simple argumentless function with no return value
type Thunk func()

// Wrap thunk in calls that lock the executing go routine to some OSThread
func (p Thunk) WithOSThread() Thunk {
  return Thunk(func() {
    rt.LockOSThread()
    defer rt.UnlockOSThread()

    p()
  })
}

// Helper for calling thunk within a separate go routine bound to a
// fixed OSThread
func (p Thunk) RunInOSThread() {
  go (p.WithOSThread())()
}

// Wrap thunk such that it sends notifi after finishing
// (May discard errors!)
func (p Thunk) Syncing(ch chan interface{}, msg interface{}) Thunk {
  return Thunk(func() {
    defer func() { ch <- msg }()
    p()
  })
}


// ******** Configuration ********

// Integer value of environment variable GOMAXPROCS if > 1, 1 otherwise
func EnvGOMAXPROCS() int {
  var maxProcs, error = strconv.Atoi(os.Getenv("GOMAXPROCS"))
  if error == nil && maxProcs > 1 {
    return maxProcs
  }
  return 1
}


// ******** Error Handling ********

// Panics with error if cond is true
func CondPanic(cond bool, error os.Error) {
  if cond {
    panic(error)
  }
}

// Deliver current errno from C.
// For this to work reliably, you must lock the executing goroutine to the
// underlying OSThread, i.e. by using GoThread!
func errno() os.Errno { return os.Errno(uint64(C.get_errno())) }

// Type of Errno() to os.Error conversion functions
type ErrnoFun func(os.Errno) os.Error

// Calls CatchError(errnoFun) iff cond is true.
// Requires that the executing go routine has been locked to an OSThread.
func CondCatchError(cond bool, errnoFun ErrnoFun) {
  if cond {
    CatchError(errnoFun)
  }
}

// Gets errno from C and converts it into an os.Error using errnoFun.
// Requires that the executing go routine has been locked to an OSThread.
func CatchError(errnoFun ErrnoFun) {
  CatchErrno(errno(), errnoFun)
}

// Converts c_errno into an os.Error using errnoFun.
// Requires that the executing go routine has been locked to an OSThread.
func CatchErrno(c_errno os.Errno, errnoFun ErrnoFun) {
  if c_errno != os.Errno(0) {
    error := errnoFun(c_errno)
    if error == nil {
      panic(os.Error(c_errno))
    } else {
      panic(error)
    }
  }
}

// Converts c_errno into an os.Error using errnoFun.
// Requires that the executing go routine has been locked to an OSThread.
func FetchError(c_errno os.Errno, errnoFun ErrnoFun) os.Error {
  if c_errno == 0 {
    return nil
  }

  error := errnoFun(c_errno)
  if error == nil {
    return os.Error(c_errno)
  }
  return error
}
