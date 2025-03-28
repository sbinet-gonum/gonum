// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mat

import (
	"fmt"
	"math"
	"math/rand/v2"
	"reflect"
	"testing"

	"gonum.org/v1/gonum/blas"
	"gonum.org/v1/gonum/blas/blas64"
)

func TestNewTriangular(t *testing.T) {
	t.Parallel()
	for i, test := range []struct {
		data []float64
		n    int
		kind TriKind
		mat  *TriDense
	}{
		{
			data: []float64{
				1, 2, 3,
				4, 5, 6,
				7, 8, 9,
			},
			n:    3,
			kind: Upper,
			mat: &TriDense{
				mat: blas64.Triangular{
					N:      3,
					Stride: 3,
					Uplo:   blas.Upper,
					Data:   []float64{1, 2, 3, 4, 5, 6, 7, 8, 9},
					Diag:   blas.NonUnit,
				},
				cap: 3,
			},
		},
	} {
		tri := NewTriDense(test.n, test.kind, test.data)
		rows, cols := tri.Dims()

		if rows != test.n {
			t.Errorf("unexpected number of rows for test %d: got: %d want: %d", i, rows, test.n)
		}
		if cols != test.n {
			t.Errorf("unexpected number of cols for test %d: got: %d want: %d", i, cols, test.n)
		}
		if !reflect.DeepEqual(tri, test.mat) {
			t.Errorf("unexpected data slice for test %d: got: %v want: %v", i, tri, test.mat)
		}
	}

	for _, kind := range []TriKind{Lower, Upper} {
		panicked, message := panics(func() { NewTriDense(3, kind, []float64{1, 2}) })
		if !panicked || message != ErrShape.Error() {
			t.Errorf("expected panic for invalid data slice length for upper=%t", kind)
		}
	}
}

func TestTriAtSet(t *testing.T) {
	t.Parallel()
	tri := &TriDense{
		mat: blas64.Triangular{
			N:      3,
			Stride: 3,
			Uplo:   blas.Upper,
			Data:   []float64{1, 2, 3, 4, 5, 6, 7, 8, 9},
			Diag:   blas.NonUnit,
		},
		cap: 3,
	}

	rows, cols := tri.Dims()

	// Check At out of bounds
	for _, row := range []int{-1, rows, rows + 1} {
		panicked, message := panics(func() { tri.At(row, 0) })
		if !panicked || message != ErrRowAccess.Error() {
			t.Errorf("expected panic for invalid row access N=%d r=%d", rows, row)
		}
	}
	for _, col := range []int{-1, cols, cols + 1} {
		panicked, message := panics(func() { tri.At(0, col) })
		if !panicked || message != ErrColAccess.Error() {
			t.Errorf("expected panic for invalid column access N=%d c=%d", cols, col)
		}
	}

	// Check Set out of bounds
	for _, row := range []int{-1, rows, rows + 1} {
		panicked, message := panics(func() { tri.SetTri(row, 0, 1.2) })
		if !panicked || message != ErrRowAccess.Error() {
			t.Errorf("expected panic for invalid row access N=%d r=%d", rows, row)
		}
	}
	for _, col := range []int{-1, cols, cols + 1} {
		panicked, message := panics(func() { tri.SetTri(0, col, 1.2) })
		if !panicked || message != ErrColAccess.Error() {
			t.Errorf("expected panic for invalid column access N=%d c=%d", cols, col)
		}
	}

	for _, st := range []struct {
		row, col int
		uplo     blas.Uplo
	}{
		{row: 2, col: 1, uplo: blas.Upper},
		{row: 1, col: 2, uplo: blas.Lower},
	} {
		tri.mat.Uplo = st.uplo
		panicked, message := panics(func() { tri.SetTri(st.row, st.col, 1.2) })
		if !panicked || message != ErrTriangleSet.Error() {
			t.Errorf("expected panic for %+v", st)
		}
	}

	for _, st := range []struct {
		row, col  int
		uplo      blas.Uplo
		orig, new float64
	}{
		{row: 2, col: 1, uplo: blas.Lower, orig: 8, new: 15},
		{row: 1, col: 2, uplo: blas.Upper, orig: 6, new: 15},
	} {
		tri.mat.Uplo = st.uplo
		if e := tri.At(st.row, st.col); e != st.orig {
			t.Errorf("unexpected value for At(%d, %d): got: %v want: %v", st.row, st.col, e, st.orig)
		}
		tri.SetTri(st.row, st.col, st.new)
		if e := tri.At(st.row, st.col); e != st.new {
			t.Errorf("unexpected value for At(%d, %d) after SetTri(%[1]d, %d, %v): got: %v want: %[3]v", st.row, st.col, st.new, e)
		}
	}
}

func TestTriDenseZero(t *testing.T) {
	t.Parallel()
	// Elements that equal 1 should be set to zero, elements that equal -1
	// should remain unchanged.
	for _, test := range []*TriDense{
		{
			mat: blas64.Triangular{
				Uplo:   blas.Upper,
				N:      4,
				Stride: 5,
				Data: []float64{
					1, 1, 1, 1, -1,
					-1, 1, 1, 1, -1,
					-1, -1, 1, 1, -1,
					-1, -1, -1, 1, -1,
				},
			},
		},
		{
			mat: blas64.Triangular{
				Uplo:   blas.Lower,
				N:      4,
				Stride: 5,
				Data: []float64{
					1, -1, -1, -1, -1,
					1, 1, -1, -1, -1,
					1, 1, 1, -1, -1,
					1, 1, 1, 1, -1,
				},
			},
		},
	} {
		dataCopy := make([]float64, len(test.mat.Data))
		copy(dataCopy, test.mat.Data)
		test.Zero()
		for i, v := range test.mat.Data {
			if dataCopy[i] != -1 && v != 0 {
				t.Errorf("Matrix not zeroed in bounds")
			}
			if dataCopy[i] == -1 && v != -1 {
				t.Errorf("Matrix zeroed out of bounds")
			}
		}
	}
}

func TestTriDiagView(t *testing.T) {
	t.Parallel()
	for cas, test := range []*TriDense{
		NewTriDense(1, Upper, []float64{1}),
		NewTriDense(2, Upper, []float64{1, 2, 0, 3}),
		NewTriDense(3, Upper, []float64{1, 2, 3, 0, 4, 5, 0, 0, 6}),
		NewTriDense(1, Lower, []float64{1}),
		NewTriDense(2, Lower, []float64{1, 2, 2, 3}),
		NewTriDense(3, Lower, []float64{1, 0, 0, 2, 3, 0, 4, 5, 6}),
	} {
		testDiagView(t, cas, test)
	}
}

func TestTriDenseCopy(t *testing.T) {
	t.Parallel()
	src := rand.NewPCG(1, 1)
	rnd := rand.New(src)
	for i := 0; i < 100; i++ {
		size := rnd.IntN(100)
		r, err := randDense(size, 0.9, src)
		if size == 0 {
			if err != ErrZeroLength {
				t.Fatalf("expected error %v: got: %v", ErrZeroLength, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		u := NewTriDense(size, true, nil)
		l := NewTriDense(size, false, nil)

		for _, typ := range []Matrix{r, (*basicMatrix)(r)} {
			for j := range u.mat.Data {
				u.mat.Data[j] = math.NaN()
				l.mat.Data[j] = math.NaN()
			}
			u.Copy(typ)
			l.Copy(typ)
			for m := 0; m < size; m++ {
				for n := 0; n < size; n++ {
					want := typ.At(m, n)
					switch {
					case m < n: // Upper triangular matrix.
						if got := u.At(m, n); got != want {
							t.Errorf("unexpected upper value for At(%d, %d) for test %d: got: %v want: %v", m, n, i, got, want)
						}
					case m == n: // Diagonal matrix.
						if got := u.At(m, n); got != want {
							t.Errorf("unexpected upper value for At(%d, %d) for test %d: got: %v want: %v", m, n, i, got, want)
						}
						if got := l.At(m, n); got != want {
							t.Errorf("unexpected diagonal value for At(%d, %d) for test %d: got: %v want: %v", m, n, i, got, want)
						}
					case m > n: // Lower triangular matrix.
						if got := l.At(m, n); got != want {
							t.Errorf("unexpected lower value for At(%d, %d) for test %d: got: %v want: %v", m, n, i, got, want)
						}
					}
				}
			}
		}
	}
}

func TestTriTriDenseCopy(t *testing.T) {
	t.Parallel()
	src := rand.NewPCG(1, 1)
	rnd := rand.New(src)
	for i := 0; i < 100; i++ {
		size := rnd.IntN(100)
		r, err := randDense(size, 1, src)
		if size == 0 {
			if err != ErrZeroLength {
				t.Fatalf("expected error %v: got: %v", ErrZeroLength, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ur := NewTriDense(size, true, nil)
		lr := NewTriDense(size, false, nil)

		ur.Copy(r)
		lr.Copy(r)

		u := NewTriDense(size, true, nil)
		u.Copy(ur)
		if !equal(u, ur) {
			t.Fatal("unexpected result for U triangle copy of U triangle: not equal")
		}

		l := NewTriDense(size, false, nil)
		l.Copy(lr)
		if !equal(l, lr) {
			t.Fatal("unexpected result for L triangle copy of L triangle: not equal")
		}

		zero(u.mat.Data)
		u.Copy(lr)
		if !isDiagonal(u) {
			t.Fatal("unexpected result for U triangle copy of L triangle: off diagonal non-zero element")
		}
		if !equalDiagonal(u, lr) {
			t.Fatal("unexpected result for U triangle copy of L triangle: diagonal not equal")
		}

		zero(l.mat.Data)
		l.Copy(ur)
		if !isDiagonal(l) {
			t.Fatal("unexpected result for L triangle copy of U triangle: off diagonal non-zero element")
		}
		if !equalDiagonal(l, ur) {
			t.Fatal("unexpected result for L triangle copy of U triangle: diagonal not equal")
		}
	}
}

func TestTriInverse(t *testing.T) {
	t.Parallel()
	rnd := rand.New(rand.NewPCG(1, 1))
	for _, kind := range []TriKind{Upper, Lower} {
		for _, n := range []int{1, 3, 5, 9} {
			data := make([]float64, n*n)
			for i := range data {
				data[i] = rnd.NormFloat64()
			}
			a := NewTriDense(n, kind, data)
			var tr TriDense
			err := tr.InverseTri(a)
			if err != nil {
				t.Errorf("Bad test: %s", err)
			}
			var d Dense
			d.Mul(a, &tr)
			if !equalApprox(eye(n), &d, 1e-8, false) {
				var diff Dense
				diff.Sub(eye(n), &d)
				t.Errorf("Tri times inverse is not identity. Norm of difference: %v", Norm(&diff, 2))
			}
		}
	}
}

func TestTriMul(t *testing.T) {
	t.Parallel()
	method := func(receiver, a, b Matrix) {
		type MulTrier interface {
			MulTri(a, b Triangular)
		}
		receiver.(MulTrier).MulTri(a.(Triangular), b.(Triangular))
	}
	denseComparison := func(receiver, a, b *Dense) {
		receiver.Mul(a, b)
	}
	legalSizeTriMul := func(ar, ac, br, bc int) bool {
		// Need both to be square and the sizes to be the same
		return ar == ac && br == bc && ar == br
	}

	// The legal types are triangles with the same TriKind.
	// legalTypesTri returns whether both input arguments are Triangular.
	legalTypes := func(a, b Matrix) bool {
		at, ok := a.(Triangular)
		if !ok {
			return false
		}
		bt, ok := b.(Triangular)
		if !ok {
			return false
		}
		_, ak := at.Triangle()
		_, bk := bt.Triangle()
		return ak == bk
	}
	legalTypesLower := func(a, b Matrix) bool {
		legal := legalTypes(a, b)
		if !legal {
			return false
		}
		_, kind := a.(Triangular).Triangle()
		r := kind == Lower
		return r
	}
	receiver := NewTriDense(3, Lower, nil)
	testTwoInput(t, "TriMul", receiver, method, denseComparison, legalTypesLower, legalSizeTriMul, 1e-14)

	legalTypesUpper := func(a, b Matrix) bool {
		legal := legalTypes(a, b)
		if !legal {
			return false
		}
		_, kind := a.(Triangular).Triangle()
		r := kind == Upper
		return r
	}
	receiver = NewTriDense(3, Upper, nil)
	testTwoInput(t, "TriMul", receiver, method, denseComparison, legalTypesUpper, legalSizeTriMul, 1e-14)
}

func TestScaleTri(t *testing.T) {
	t.Parallel()
	for _, f := range []float64{0.5, 1, 3} {
		method := func(receiver, a Matrix) {
			type ScaleTrier interface {
				ScaleTri(f float64, a Triangular)
			}
			rd := receiver.(ScaleTrier)
			rd.ScaleTri(f, a.(Triangular))
		}
		denseComparison := func(receiver, a *Dense) {
			receiver.Scale(f, a)
		}
		testOneInput(t, "ScaleTriUpper", NewTriDense(3, Upper, nil), method, denseComparison, legalTypeTriUpper, isSquare, 1e-14)
		testOneInput(t, "ScaleTriLower", NewTriDense(3, Lower, nil), method, denseComparison, legalTypeTriLower, isSquare, 1e-14)
	}
}

func TestCopySymIntoTriangle(t *testing.T) {
	t.Parallel()
	nan := math.NaN()
	for tc, test := range []struct {
		n     int
		sUplo blas.Uplo
		s     []float64

		tUplo TriKind
		want  []float64
	}{
		{
			n:     3,
			sUplo: blas.Upper,
			s: []float64{
				1, 2, 3,
				nan, 4, 5,
				nan, nan, 6,
			},
			tUplo: Upper,
			want: []float64{
				1, 2, 3,
				0, 4, 5,
				0, 0, 6,
			},
		},
		{
			n:     3,
			sUplo: blas.Lower,
			s: []float64{
				1, nan, nan,
				2, 3, nan,
				4, 5, 6,
			},
			tUplo: Upper,
			want: []float64{
				1, 2, 4,
				0, 3, 5,
				0, 0, 6,
			},
		},
		{
			n:     3,
			sUplo: blas.Upper,
			s: []float64{
				1, 2, 3,
				nan, 4, 5,
				nan, nan, 6,
			},
			tUplo: Lower,
			want: []float64{
				1, 0, 0,
				2, 4, 0,
				3, 5, 6,
			},
		},
		{
			n:     3,
			sUplo: blas.Lower,
			s: []float64{
				1, nan, nan,
				2, 3, nan,
				4, 5, 6,
			},
			tUplo: Lower,
			want: []float64{
				1, 0, 0,
				2, 3, 0,
				4, 5, 6,
			},
		},
	} {
		n := test.n
		s := NewSymDense(n, test.s)
		// For the purpose of the test, break the assumption that
		// symmetric is stored in the upper triangle (only when S is
		// RawSymmetricer).
		s.mat.Uplo = test.sUplo

		t1 := NewTriDense(n, test.tUplo, nil)
		copySymIntoTriangle(t1, s)

		equal := true
	loop1:
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if t1.At(i, j) != test.want[i*n+j] {
					equal = false
					break loop1
				}
			}
		}
		if !equal {
			t.Errorf("Case %v: unexpected T when S is RawSymmetricer", tc)
		}

		if test.sUplo == blas.Lower {
			continue
		}

		sb := (basicSymmetric)(*s)
		t2 := NewTriDense(n, test.tUplo, nil)
		copySymIntoTriangle(t2, &sb)
		equal = true
	loop2:
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if t1.At(i, j) != test.want[i*n+j] {
					equal = false
					break loop2
				}
			}
		}
		if !equal {
			t.Errorf("Case %v: unexpected T when S is not RawSymmetricer", tc)
		}
	}
}

func TestTriSliceTri(t *testing.T) {
	t.Parallel()
	rnd := rand.New(rand.NewPCG(1, 1))
	for cas, test := range []struct {
		n, start1, span1, start2, span2 int
	}{
		{10, 0, 10, 0, 10},
		{10, 0, 8, 0, 8},
		{10, 2, 8, 0, 6},
		{10, 2, 7, 4, 2},
		{10, 2, 6, 0, 5},
		{10, 2, 3, 1, 7},
	} {
		n := test.n
		for _, kind := range []TriKind{Upper, Lower} {
			tri := NewTriDense(n, kind, nil)
			if kind == Upper {
				for i := 0; i < n; i++ {
					for j := i; j < n; j++ {
						tri.SetTri(i, j, rnd.Float64())
					}
				}
			} else {
				for i := 0; i < n; i++ {
					for j := 0; j <= i; j++ {
						tri.SetTri(i, j, rnd.Float64())
					}
				}
			}

			start1 := test.start1
			span1 := test.span1
			v1 := tri.SliceTri(start1, start1+span1).(*TriDense)
			if kind == Upper {
				for i := 0; i < span1; i++ {
					for j := i; j < span1; j++ {
						if v1.At(i, j) != tri.At(start1+i, start1+j) {
							t.Errorf("Case %d,upper: view mismatch at %v,%v", cas, i, j)
						}
					}
				}
			} else {
				for i := 0; i < span1; i++ {
					for j := 0; j <= i; j++ {
						if v1.At(i, j) != tri.At(start1+i, start1+j) {
							t.Errorf("Case %d,lower: view mismatch at %v,%v", cas, i, j)
						}
					}
				}
			}

			start2 := test.start2
			span2 := test.span2
			v2 := v1.SliceTri(start2, start2+span2).(*TriDense)
			if kind == Upper {
				for i := 0; i < span2; i++ {
					for j := i; j < span2; j++ {
						if v2.At(i, j) != tri.At(start1+start2+i, start1+start2+j) {
							t.Errorf("Case %d,upper: second view mismatch at %v,%v", cas, i, j)
						}
					}
				}
			} else {
				for i := 0; i < span1; i++ {
					for j := 0; j <= i; j++ {
						if v1.At(i, j) != tri.At(start1+i, start1+j) {
							t.Errorf("Case %d,lower: second view mismatch at %v,%v", cas, i, j)
						}
					}
				}
			}

			v2.SetTri(span2-1, span2-1, -123.45)
			if tri.At(start1+start2+span2-1, start1+start2+span2-1) != -123.45 {
				t.Errorf("Case %d: write to view not reflected in original", cas)
			}
		}
	}
}

var triSumForBench float64

func BenchmarkTriSum(b *testing.B) {
	rnd := rand.New(rand.NewPCG(1, 1))
	for n := 100; n <= 1600; n *= 2 {
		a := randTriDense(n, rnd)
		b.Run(fmt.Sprintf("BenchmarkTriSum%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				triSumForBench = Sum(a)
			}
		})
	}
}

var triProductForBench *TriDense

func BenchmarkTriMul(b *testing.B) {
	rnd := rand.New(rand.NewPCG(1, 1))
	for n := 100; n <= 1600; n *= 2 {
		triProductForBench = NewTriDense(n, Upper, nil)
		a := randTriDense(n, rnd)
		c := randTriDense(n, rnd)
		b.Run(fmt.Sprintf("BenchmarkTriMul%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				triProductForBench.MulTri(a, c)
			}
		})
	}
}

func BenchmarkTriMulDiag(b *testing.B) {
	rnd := rand.New(rand.NewPCG(1, 1))
	for n := 100; n <= 1600; n *= 2 {
		triProductForBench = NewTriDense(n, Upper, nil)
		a := randTriDense(n, rnd)
		c := randDiagDense(n, rnd)
		b.Run(fmt.Sprintf("BenchmarkTriMulDiag%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				triProductForBench.MulTri(a, c)
			}
		})
	}
}

func BenchmarkTriMul2Diag(b *testing.B) {
	rnd := rand.New(rand.NewPCG(1, 1))
	for n := 100; n <= 1600; n *= 2 {
		triProductForBench = NewTriDense(n, Upper, nil)
		a := randDiagDense(n, rnd)
		c := randDiagDense(n, rnd)
		b.Run(fmt.Sprintf("BenchmarkTriMul2Diag%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				triProductForBench.MulTri(a, c)
			}
		})
	}
}

func randTriDense(size int, rnd *rand.Rand) *TriDense {
	t := NewTriDense(size, Upper, nil)
	for i := 0; i < size; i++ {
		for j := i; j < size; j++ {
			t.SetTri(i, j, rnd.Float64())
		}
	}
	return t
}
