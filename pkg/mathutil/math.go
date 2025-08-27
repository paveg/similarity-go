package mathutil

import "cmp"

// Min returns the minimum of two comparable values.
func Min[T cmp.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum of two comparable values.
func Max[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Abs returns the absolute value of a signed numeric value.
func Abs[T interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64
}](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

// MinInt is a convenience function for integer minimum.
// Deprecated: Use Min[int] instead.
func MinInt(a, b int) int {
	return Min(a, b)
}

// MaxInt is a convenience function for integer maximum.
// Deprecated: Use Max[int] instead.
func MaxInt(a, b int) int {
	return Max(a, b)
}

// AbsInt is a convenience function for integer absolute value.
// Deprecated: Use Abs[int] instead.
func AbsInt(x int) int {
	return Abs(x)
}
