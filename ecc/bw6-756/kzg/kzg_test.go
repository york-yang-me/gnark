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

package kzg

import (
	"crypto/sha256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bw6-756"
	"github.com/consensys/gnark-crypto/ecc/bw6-756/fr"
	"github.com/consensys/gnark-crypto/ecc/bw6-756/fr/fft"

	"github.com/consensys/gnark-crypto/utils"
)

// Test SRS re-used across tests of the KZG scheme
var testSrs *SRS
var bAlpha *big.Int

func init() {
	const srsSize = 230
	bAlpha = new(big.Int).SetInt64(42) // randomise ?
	testSrs, _ = NewSRS(ecc.NextPowerOfTwo(srsSize), bAlpha)
}

func TestToLagrangeG1(t *testing.T) {
	assert := require.New(t)

	const size = 32

	// convert the test SRS to Lagrange form
	lagrange, err := ToLagrangeG1(testSrs.Pk.G1[:size])
	assert.NoError(err)

	// generate the Lagrange SRS manually and compare
	w, err := fr.Generator(uint64(size))
	assert.NoError(err)

	var li, n, d, one, acc, alpha fr.Element
	alpha.SetBigInt(bAlpha)
	li.SetUint64(uint64(size)).Inverse(&li)
	one.SetOne()
	n.Exp(alpha, big.NewInt(int64(size))).Sub(&n, &one)
	d.Sub(&alpha, &one)
	li.Mul(&li, &n).Div(&li, &d)
	expectedSrsLagrange := make([]bw6756.G1Affine, size)
	_, _, g1Gen, _ := bw6756.Generators()
	var s big.Int
	acc.SetOne()
	for i := 0; i < size; i++ {
		li.BigInt(&s)
		expectedSrsLagrange[i].ScalarMultiplication(&g1Gen, &s)

		li.Mul(&li, &w).Mul(&li, &d)
		acc.Mul(&acc, &w)
		d.Sub(&alpha, &acc)
		li.Div(&li, &d)
	}

	for i := 0; i < size; i++ {
		assert.True(expectedSrsLagrange[i].Equal(&lagrange[i]), "error lagrange conversion")
	}
}

func TestCommitLagrange(t *testing.T) {

	assert := require.New(t)

	// sample a sparse polynomial (here in Lagrange form)
	size := 64
	pol := make([]fr.Element, size)
	pol[0].SetRandom()
	for i := 0; i < size; i = i + 8 {
		pol[i].SetRandom()
	}

	// commitment using Lagrange SRS
	lagrange, err := ToLagrangeG1(testSrs.Pk.G1[:size])
	assert.NoError(err)
	var pkLagrange ProvingKey
	pkLagrange.G1 = lagrange

	digestLagrange, err := Commit(pol, pkLagrange)
	assert.NoError(err)

	// commitment using canonical SRS
	d := fft.NewDomain(uint64(size))
	d.FFTInverse(pol, fft.DIF)
	fft.BitReverse(pol)
	digestCanonical, err := Commit(pol, testSrs.Pk)
	assert.NoError(err)

	// compare the results
	assert.True(digestCanonical.Equal(&digestLagrange), "error CommitLagrange")
}

func TestDividePolyByXminusA(t *testing.T) {

	const pSize = 230

	// build random polynomial
	pol := make([]fr.Element, pSize)
	pol[0].SetRandom()
	for i := 1; i < pSize; i++ {
		pol[i] = pol[i-1]
	}

	// evaluate the polynomial at a random point
	var point fr.Element
	point.SetRandom()
	evaluation := eval(pol, point)

	// probabilistic test (using Schwartz Zippel lemma, evaluation at one point is enough)
	var randPoint, xminusa fr.Element
	randPoint.SetRandom()
	polRandpoint := eval(pol, randPoint)
	polRandpoint.Sub(&polRandpoint, &evaluation) // f(rand)-f(point)

	// compute f-f(a)/x-a
	// h re-uses the memory of pol
	h := dividePolyByXminusA(pol, evaluation, point)

	if len(h) != 229 {
		t.Fatal("inconsistent size of quotient")
	}

	hRandPoint := eval(h, randPoint)
	xminusa.Sub(&randPoint, &point) // rand-point

	// f(rand)-f(point)	==? h(rand)*(rand-point)
	hRandPoint.Mul(&hRandPoint, &xminusa)

	if !hRandPoint.Equal(&polRandpoint) {
		t.Fatal("Error f-f(a)/x-a")
	}
}

func TestSerializationSRS(t *testing.T) {
	// create a SRS
	srs, err := NewSRS(64, new(big.Int).SetInt64(42))
	assert.NoError(t, err)
	t.Run("proving key round-trip", utils.SerializationRoundTrip(&srs.Pk))
	t.Run("proving key raw round-trip", utils.SerializationRoundTripRaw(&srs.Pk))
	t.Run("verifying key round-trip", utils.SerializationRoundTrip(&srs.Vk))
	t.Run("whole SRS round-trip", utils.SerializationRoundTrip(srs))
}

func TestCommit(t *testing.T) {

	// create a polynomial
	f := make([]fr.Element, 60)
	for i := 0; i < 60; i++ {
		f[i].SetRandom()
	}

	// commit using the method from KZG
	_kzgCommit, err := Commit(f, testSrs.Pk)
	if err != nil {
		t.Fatal(err)
	}
	var kzgCommit bw6756.G1Affine
	kzgCommit.Unmarshal(_kzgCommit.Marshal())

	// check commitment using manual commit
	var x fr.Element
	x.SetString("42")
	fx := eval(f, x)
	var fxbi big.Int
	fx.BigInt(&fxbi)
	var manualCommit bw6756.G1Affine
	manualCommit.Set(&testSrs.Vk.G1)
	manualCommit.ScalarMultiplication(&manualCommit, &fxbi)

	// compare both results
	if !kzgCommit.Equal(&manualCommit) {
		t.Fatal("error KZG commitment")
	}

}

func TestVerifySinglePoint(t *testing.T) {

	// create a polynomial
	f := randomPolynomial(60)

	// commit the polynomial
	digest, err := Commit(f, testSrs.Pk)
	if err != nil {
		t.Fatal(err)
	}

	// compute opening proof at a random point
	var point fr.Element
	point.SetString("4321")
	proof, err := Open(f, point, testSrs.Pk)
	if err != nil {
		t.Fatal(err)
	}

	// verify the claimed valued
	expected := eval(f, point)
	if !proof.ClaimedValue.Equal(&expected) {
		t.Fatal("inconsistent claimed value")
	}

	// verify correct proof
	err = Verify(&digest, &proof, point, testSrs.Vk)
	if err != nil {
		t.Fatal(err)
	}

	{
		// verify wrong proof
		proof.ClaimedValue.Double(&proof.ClaimedValue)
		err = Verify(&digest, &proof, point, testSrs.Vk)
		if err == nil {
			t.Fatal("verifying wrong proof should have failed")
		}
	}
	{
		// verify wrong proof with quotient set to zero
		// see https://cryptosubtlety.medium.com/00-8d4adcf4d255
		proof.H.X.SetZero()
		proof.H.Y.SetZero()
		err = Verify(&digest, &proof, point, testSrs.Vk)
		if err == nil {
			t.Fatal("verifying wrong proof should have failed")
		}
	}
}

func TestVerifySinglePointQuickSRS(t *testing.T) {

	size := 64
	srs, err := NewSRS(64, big.NewInt(-1))
	if err != nil {
		t.Fatal(err)
	}

	// random polynomial
	p := make([]fr.Element, size)
	for i := 0; i < size; i++ {
		p[i].SetRandom()
	}

	// random value
	var x fr.Element
	x.SetRandom()

	// verify valid proof
	d, err := Commit(p, srs.Pk)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := Open(p, x, srs.Pk)
	if err != nil {
		t.Fatal(err)
	}
	err = Verify(&d, &proof, x, srs.Vk)
	if err != nil {
		t.Fatal(err)
	}

	// verify wrong proof
	proof.ClaimedValue.SetRandom()
	err = Verify(&d, &proof, x, srs.Vk)
	if err == nil {
		t.Fatal(err)
	}

}

func TestBatchVerifySinglePoint(t *testing.T) {

	size := 40

	// create polynomials
	f := make([][]fr.Element, 10)
	for i := range f {
		f[i] = randomPolynomial(size)
	}

	// commit the polynomials
	digests := make([]Digest, len(f))
	for i := range f {
		digests[i], _ = Commit(f[i], testSrs.Pk)

	}

	// pick a hash function
	hf := sha256.New()

	// compute opening proof at a random point
	var point fr.Element
	point.SetString("4321")
	proof, err := BatchOpenSinglePoint(f, digests, point, hf, testSrs.Pk)
	if err != nil {
		t.Fatal(err)
	}

	var salt fr.Element
	salt.SetRandom()
	proofExtendedTranscript, err := BatchOpenSinglePoint(f, digests, point, hf, testSrs.Pk, salt.Marshal())
	if err != nil {
		t.Fatal(err)
	}

	// verify the claimed values
	for i := range f {
		expectedClaim := eval(f[i], point)
		if !expectedClaim.Equal(&proof.ClaimedValues[i]) {
			t.Fatal("inconsistent claimed values")
		}
	}

	// verify correct proof
	err = BatchVerifySinglePoint(digests, &proof, point, hf, testSrs.Vk)
	if err != nil {
		t.Fatal(err)
	}

	// verify correct proof with extended transcript
	err = BatchVerifySinglePoint(digests, &proofExtendedTranscript, point, hf, testSrs.Vk, salt.Marshal())
	if err != nil {
		t.Fatal(err)
	}

	{
		// verify wrong proof
		proof.ClaimedValues[0].Double(&proof.ClaimedValues[0])
		err = BatchVerifySinglePoint(digests, &proof, point, hf, testSrs.Vk)
		if err == nil {
			t.Fatal("verifying wrong proof should have failed")
		}
	}
	{
		// verify wrong proof with quotient set to zero
		// see https://cryptosubtlety.medium.com/00-8d4adcf4d255
		proof.H.X.SetZero()
		proof.H.Y.SetZero()
		err = BatchVerifySinglePoint(digests, &proof, point, hf, testSrs.Vk)
		if err == nil {
			t.Fatal("verifying wrong proof should have failed")
		}
	}
}

func TestBatchVerifyMultiPoints(t *testing.T) {

	// create polynomials
	f := make([][]fr.Element, 10)
	for i := 0; i < 10; i++ {
		f[i] = randomPolynomial(40)
	}

	// commit the polynomials
	digests := make([]Digest, 10)
	for i := 0; i < 10; i++ {
		digests[i], _ = Commit(f[i], testSrs.Pk)
	}

	// pick a hash function
	hf := sha256.New()

	// compute 2 batch opening proofs at 2 random points
	points := make([]fr.Element, 2)
	batchProofs := make([]BatchOpeningProof, 2)
	points[0].SetRandom()
	batchProofs[0], _ = BatchOpenSinglePoint(f[:5], digests[:5], points[0], hf, testSrs.Pk)
	points[1].SetRandom()
	batchProofs[1], _ = BatchOpenSinglePoint(f[5:], digests[5:], points[1], hf, testSrs.Pk)

	// fold the 2 batch opening proofs
	proofs := make([]OpeningProof, 2)
	foldedDigests := make([]Digest, 2)
	proofs[0], foldedDigests[0], _ = FoldProof(digests[:5], &batchProofs[0], points[0], hf)
	proofs[1], foldedDigests[1], _ = FoldProof(digests[5:], &batchProofs[1], points[1], hf)

	// check that the individual batch proofs are correct
	err := Verify(&foldedDigests[0], &proofs[0], points[0], testSrs.Vk)
	if err != nil {
		t.Fatal(err)
	}
	err = Verify(&foldedDigests[1], &proofs[1], points[1], testSrs.Vk)
	if err != nil {
		t.Fatal(err)
	}

	// batch verify correct folded proofs
	err = BatchVerifyMultiPoints(foldedDigests, proofs, points, testSrs.Vk)
	if err != nil {
		t.Fatal(err)
	}

	{
		// batch verify tampered folded proofs
		proofs[0].ClaimedValue.Double(&proofs[0].ClaimedValue)

		err = BatchVerifyMultiPoints(foldedDigests, proofs, points, testSrs.Vk)
		if err == nil {
			t.Fatal(err)
		}
	}
	{
		// batch verify tampered folded proofs with quotients set to infinity
		// see https://cryptosubtlety.medium.com/00-8d4adcf4d255
		proofs[0].H.X.SetZero()
		proofs[0].H.Y.SetZero()
		proofs[1].H.X.SetZero()
		proofs[1].H.Y.SetZero()
		err = BatchVerifyMultiPoints(foldedDigests, proofs, points, testSrs.Vk)
		if err == nil {
			t.Fatal(err)
		}
	}

}

const benchSize = 1 << 16

func BenchmarkSRSGen(b *testing.B) {

	b.Run("real SRS", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			NewSRS(ecc.NextPowerOfTwo(benchSize), new(big.Int).SetInt64(42))
		}
	})
	b.Run("quick SRS", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			NewSRS(ecc.NextPowerOfTwo(benchSize), big.NewInt(-1))
		}
	})
}

func BenchmarkKZGCommit(b *testing.B) {

	b.Run("real SRS", func(b *testing.B) {
		srs, err := NewSRS(ecc.NextPowerOfTwo(benchSize), new(big.Int).SetInt64(42))
		assert.NoError(b, err)
		// random polynomial
		p := randomPolynomial(benchSize / 2)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = Commit(p, srs.Pk)
		}
	})
	b.Run("quick SRS", func(b *testing.B) {
		srs, err := NewSRS(ecc.NextPowerOfTwo(benchSize), big.NewInt(-1))
		assert.NoError(b, err)
		// random polynomial
		p := randomPolynomial(benchSize / 2)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = Commit(p, srs.Pk)
		}
	})
}

func BenchmarkDivideByXMinusA(b *testing.B) {
	const pSize = 1 << 22

	// build random polynomial
	pol := make([]fr.Element, pSize)
	pol[0].SetRandom()
	for i := 1; i < pSize; i++ {
		pol[i] = pol[i-1]
	}
	var a, fa fr.Element
	a.SetRandom()
	fa.SetRandom()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dividePolyByXminusA(pol, fa, a)
		pol = pol[:pSize]
		pol[pSize-1] = pol[0]
	}
}

func BenchmarkKZGOpen(b *testing.B) {
	srs, err := NewSRS(ecc.NextPowerOfTwo(benchSize), new(big.Int).SetInt64(42))
	assert.NoError(b, err)

	// random polynomial
	p := randomPolynomial(benchSize / 2)
	var r fr.Element
	r.SetRandom()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Open(p, r, srs.Pk)
	}
}

func BenchmarkKZGVerify(b *testing.B) {
	srs, err := NewSRS(ecc.NextPowerOfTwo(benchSize), new(big.Int).SetInt64(42))
	assert.NoError(b, err)

	// random polynomial
	p := randomPolynomial(benchSize / 2)
	var r fr.Element
	r.SetRandom()

	// commit
	comm, err := Commit(p, srs.Pk)
	assert.NoError(b, err)

	// open
	openingProof, err := Open(p, r, srs.Pk)
	assert.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Verify(&comm, &openingProof, r, srs.Vk)
	}
}

func BenchmarkKZGBatchOpen10(b *testing.B) {
	srs, err := NewSRS(ecc.NextPowerOfTwo(benchSize), new(big.Int).SetInt64(42))
	assert.NoError(b, err)

	// 10 random polynomials
	var ps [10][]fr.Element
	for i := 0; i < 10; i++ {
		ps[i] = randomPolynomial(benchSize / 2)
	}

	// commitments
	var commitments [10]Digest
	for i := 0; i < 10; i++ {
		commitments[i], _ = Commit(ps[i], srs.Pk)
	}

	// pick a hash function
	hf := sha256.New()

	var r fr.Element
	r.SetRandom()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BatchOpenSinglePoint(ps[:], commitments[:], r, hf, srs.Pk)
	}
}

func BenchmarkKZGBatchVerify10(b *testing.B) {
	srs, err := NewSRS(ecc.NextPowerOfTwo(benchSize), new(big.Int).SetInt64(42))
	if err != nil {
		b.Fatal(err)
	}

	// 10 random polynomials
	var ps [10][]fr.Element
	for i := 0; i < 10; i++ {
		ps[i] = randomPolynomial(benchSize / 2)
	}

	// commitments
	var commitments [10]Digest
	for i := 0; i < 10; i++ {
		commitments[i], _ = Commit(ps[i], srs.Pk)
	}

	// pick a hash function
	hf := sha256.New()

	var r fr.Element
	r.SetRandom()

	proof, err := BatchOpenSinglePoint(ps[:], commitments[:], r, hf, srs.Pk)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BatchVerifySinglePoint(commitments[:], &proof, r, hf, testSrs.Vk)
	}
}

func randomPolynomial(size int) []fr.Element {
	f := make([]fr.Element, size)
	for i := 0; i < size; i++ {
		f[i].SetRandom()
	}
	return f
}

func BenchmarkToLagrangeG1(b *testing.B) {
	const size = 1 << 14

	var samplePoints [size]bw6756.G1Affine
	fillBenchBasesG1(samplePoints[:])
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := ToLagrangeG1(samplePoints[:]); err != nil {
			b.Fatal(err)
		}
	}
}

func fillBenchBasesG1(samplePoints []bw6756.G1Affine) {
	var r big.Int
	r.SetString("340444420969191673093399857471996460938405", 10)
	samplePoints[0].ScalarMultiplication(&samplePoints[0], &r)

	one := samplePoints[0].X
	one.SetOne()

	for i := 1; i < len(samplePoints); i++ {
		samplePoints[i].X.Add(&samplePoints[i-1].X, &one)
		samplePoints[i].Y.Sub(&samplePoints[i-1].Y, &one)
	}
}
