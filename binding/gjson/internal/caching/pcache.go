/*
 * Copyright 2021 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package caching

import (
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/bytedance/go-tagexpr/v2/binding/gjson/internal/rt"
)

/** Program Map **/

const (
	_LoadFactor   = 0.5
	_InitCapacity = 4096 // must be a power of 2
)

type _ProgramMap struct {
	n uint64
	m uint32
	b []_ProgramEntry
}

type _ProgramEntry struct {
	vt *rt.GoType
	fn interface{}
}

func newProgramMap() *_ProgramMap {
	return &_ProgramMap{
		n: 0,
		m: _InitCapacity - 1,
		b: make([]_ProgramEntry, _InitCapacity),
	}
}

func (mips *_ProgramMap) copy() *_ProgramMap {
	fork := &_ProgramMap{
		n: mips.n,
		m: mips.m,
		b: make([]_ProgramEntry, len(mips.b)),
	}
	for i, f := range mips.b {
		fork.b[i] = f
	}
	return fork
}

func (mips *_ProgramMap) get(vt *rt.GoType) interface{} {
	i := mips.m + 1
	p := vt.Hash & mips.m

	/* linear probing */
	for ; i > 0; i-- {
		if b := mips.b[p]; b.vt == vt {
			return b.fn
		} else {
			p = (p + 1) & mips.m
		}
	}

	/* not found */
	return nil
}

func (mips *_ProgramMap) add(vt *rt.GoType, fn interface{}) *_ProgramMap {
	p := mips.copy()
	f := float64(atomic.LoadUint64(&p.n)+1) / float64(p.m+1)

	/* check for load factor */
	if f > _LoadFactor {
		p = p.rehash()
	}

	/* insert the value */
	p.insert(vt, fn)
	return p
}

func (mips *_ProgramMap) rehash() *_ProgramMap {
	c := (mips.m + 1) << 1
	r := &_ProgramMap{m: c - 1, b: make([]_ProgramEntry, int(c))}

	/* rehash every entry */
	for i := uint32(0); i <= mips.m; i++ {
		if b := mips.b[i]; b.vt != nil {
			r.insert(b.vt, b.fn)
		}
	}

	/* rebuild successful */
	return r
}

func (mips *_ProgramMap) insert(vt *rt.GoType, fn interface{}) {
	h := vt.Hash
	p := h & mips.m

	/* linear probing */
	for i := uint32(0); i <= mips.m; i++ {
		if b := &mips.b[p]; b.vt != nil {
			p += 1
			p &= mips.m
		} else {
			b.vt = vt
			b.fn = fn
			atomic.AddUint64(&mips.n, 1)
			return
		}
	}

	/* should never happen */
	panic("no available slots")
}

/** RCU Program Cache **/

type ProgramCache struct {
	m sync.Mutex
	p unsafe.Pointer
}

func CreateProgramCache() *ProgramCache {
	return &ProgramCache{
		m: sync.Mutex{},
		p: unsafe.Pointer(newProgramMap()),
	}
}

func (c *ProgramCache) Get(vt *rt.GoType) interface{} {
	return (*_ProgramMap)(atomic.LoadPointer(&c.p)).get(vt)
}

func (c *ProgramCache) Compute(vt *rt.GoType, compute func(*rt.GoType) (interface{}, error)) (interface{}, error) {
	var err error
	var val interface{}

	/* use defer to prevent inlining of this function */
	c.m.Lock()
	defer c.m.Unlock()

	/* double check with write lock held */
	if val = c.Get(vt); val != nil {
		return val, nil
	}

	/* compute the value */
	if val, err = compute(vt); err != nil {
		return nil, err
	}

	/* update the RCU cache */
	atomic.StorePointer(&c.p, unsafe.Pointer((*_ProgramMap)(atomic.LoadPointer(&c.p)).add(vt, val)))
	return val, nil
}
