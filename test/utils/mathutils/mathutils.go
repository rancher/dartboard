package mathutils

func NearestMultiple(multiple int, n int) int {
	if multiple > n {
		return multiple
	}
	n = n + multiple/2
	n = n - (n % multiple)
	return n
}
