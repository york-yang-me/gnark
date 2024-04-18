// Copyright 2020 Consensys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by consensys/gnark-crypto DO NOT EDIT

package hash_to_field

import (
	"fmt"
	"hash"

	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
)

type wrappedHashToField struct {
	domain []byte
	toHash []byte
}

// New returns a new hasher instance which uses [fr.Hash] to hash all the
// written bytes to a field element, returning the byte representation of the
// field element. The domain separator is passed as-is to hashing method.
func New(domainSeparator []byte) hash.Hash {
	return &wrappedHashToField{
		domain: append([]byte{}, domainSeparator...), // copy in case the argument is modified
	}
}

func (w *wrappedHashToField) Write(p []byte) (n int, err error) {
	w.toHash = append(w.toHash, p...)
	return len(p), nil
}

func (w *wrappedHashToField) Sum(b []byte) []byte {
	res, err := fr.Hash(w.toHash, w.domain, 1)
	if err != nil {
		// we want to follow the interface, cannot return error and have to panic
		// but by default the method shouldn't return an error internally
		panic(fmt.Sprintf("native field to hash: %v", err))
	}
	bts := res[0].Bytes()
	return append(b, bts[:]...)
}

func (w *wrappedHashToField) Reset() {
	w.toHash = nil
}

func (w *wrappedHashToField) Size() int {
	return fr.Bytes
}

func (w *wrappedHashToField) BlockSize() int {
	return fr.Bytes
}
