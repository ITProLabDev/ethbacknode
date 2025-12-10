package address

import "github.com/ITProLabDev/ethbacknode/tools/log"

func (p *Manager) checkFreeAddressPool() {
	var totalAddresses, freeAddresses int
	p.mux.RLock()
	for _, v := range p.allAddresses {
		totalAddresses++
		if !v.Subscribed {
			freeAddresses++
		}
	}
	p.mux.RUnlock()
	if p.config.Debug {
		log.Debug("* Total addresses in pool:", totalAddresses)
		log.Debug("* Free addresses in pool:", freeAddresses)
	}
	if p.config.EnableAddressGenerate {
		if totalAddresses == 0 {
			p.refillFreeAddressPool(p.config.GeneratePoolUpTo)
		} else if freeAddresses < p.config.MinFreePoolSize {
			p.refillFreeAddressPool(p.config.GeneratePoolUpTo - freeAddresses)
		}
	}
}

func (p *Manager) refillFreeAddressPool(refillAmount int) {
	p.mux.Lock()
	defer p.mux.Unlock()
	for i := 0; i < refillAmount; i++ {
		var err error
		var newAddressRecord *Address
		if p.config.Bip39Support {
			newAddressRecord, err = p.GenerateBit44Address()
		} else {
			newAddressRecord, err = p.createNewAddress()
		}
		if err != nil {
			log.Error("Can not generate address:", err)
			return
		}
		p.allAddresses[newAddressRecord.Address] = newAddressRecord
		p.freeAddresses[newAddressRecord.Address] = newAddressRecord
		err = p.db.Save(newAddressRecord)
		if err != nil {
			log.Error("Can not save new address to pool:", err)
			return
		}
	}
	if p.config.Debug {
		log.Debug("* Pool refilled with", refillAmount, "new addresses")
		//log.Dump(p.allAddresses)
	}
	go p.updatePool()
}
