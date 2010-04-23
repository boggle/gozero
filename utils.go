package pin
     
import "os"   
import "sync"               
import "runtime"

// Pin

type Pin struct {
	fixed bool
	pinid uint
	
	sync.RWMutex		
}
     
func NewPin() *Pin {
	var pin = &Pin{fixed: false, pinid: 0}   
	runtime.SetFinalizer(pin, forceUnpin)
	return pin
}                            
                 
var alreadyPinned = os.NewError("OSThread already pinned")
                 
func (p *Pin) Pin() uint {
	p.Lock()        	
	defer p.Unlock()
	
	if (p.fixed) {
		panic(alreadyPinned)
	} else {          
		p.pinid++  
		runtime.LockOSThread()
		p.fixed = true
	}
	return p.pinid
}          
      
func forceUnpin(p *Pin) {
	p.Lock()
	defer p.Unlock()
	
	p.pinid++   
	runtime.UnlockOSThread()
	p.fixed = false
}                   
            
var notPinned = os.NewError("OSThread not pinned")

func (p *Pin) Unpin() {
	p.Lock()
	defer p.Unlock()
	
	if (p.fixed) {     
		p.pinid++   
		runtime.UnlockOSThread()
		p.fixed = false
	} else {
    	panic(notPinned)
	}
}                   
