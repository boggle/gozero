package zmq

import "os"
import rt "runtime"
import "sync"
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

// Reference counter interface
type RefC interface {
  Incr()
  Decr()
}

// Sample RefC implementation
type refC struct {
  lock  sync.Mutex
  count uint32
  thunk Thunk
}

var ErrRefC = os.NewError("RefC exhausted")

// Create new reference counter that will call thunk when done
// (Instantly spawns a goroutine with thunk)
func (p Thunk) RefC(initialCount uint32) RefC {
  if initialCount == 0 {
    // Somewhat pointless yet valid
    ret := &refC{count: 0, thunk: nil}
    go p()
    return ret
  }
  return &refC{count: initialCount, thunk: p}
}


// Increment RefC, panic if already 0
func (p *refC) Incr() {
  p.lock.Lock()
  defer p.lock.Unlock()

  if p.count == 0 {
    panic(ErrRefC)
  }
  p.count++
}

// Decrement RefC, calling thunk if 0 is reached.
// Panics if already at 0.
func (p *refC) Decr() {
  p.lock.Lock()
  defer p.lock.Unlock()

  switch p.count {
  case 0:
    panic(ErrRefC)
  case 1:
    thunk := p.thunk
    p.thunk = nil
    p.count = 0
    go thunk()
  default:
    p.count--
  }
}

// Create new reference counter that will call thunk when done,
// followed by sending p over chan
// (Instantly spawns a goroutine with thunk)
func (p Thunk) SyncingRefC(initialCount uint32, ch chan interface{}, msg interface{}) RefC {
  return (p.Syncing(ch, msg)).RefC(initialCount)
}


// ******** Closeable ********

// Interface for all artefacts that initially are open and later
// may be closed/terminated
type Closeable interface {
  Close()
}

// Returns RefC that triggers closeable.Close and channel that will
// deliver closeable value afer the RefC went down to 0
func SyncOnClose(c Closeable, initCount uint32) (refc RefC, ch chan interface{}) {
  ch = make(chan interface{})
  refc = Thunk(func() { c.Close() }).SyncingRefC(initCount, ch, c)
  return
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
  c_errno := errno()
  if c_errno != os.Errno(0) {
    error := errnoFun(c_errno)
    if error == nil {
      panic(os.Error(c_errno))
    } else {
      panic(error)
    }
  }
}

// {}
