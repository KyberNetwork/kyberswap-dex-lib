package bruteforce

import (
	"sort"
)

const DefaultMaxNumSplitBruteForce = 3

// nextPermutation generates the next permutation of the
// sortable collection x in lexical order.  It returns false
// if the permutations are exhausted.
//
// Knuth, Donald (2011), "Section 7.2.1.2: Generating All Permutations",
// The Art of Computer Programming, volume 4A.
func nextPermutation(x sort.Interface) bool {
	n := x.Len() - 1
	if n < 1 {
		return false
	}
	j := n - 1
	for ; !x.Less(j, j+1); j-- {
		if j == 0 {
			return false
		}
	}
	l := n
	for !x.Less(j, l) {
		l--
	}
	x.Swap(j, l)
	for k, l := j+1, n; k < l; {
		x.Swap(k, l)
		k++
		l--
	}
	return true
}

// tryElementSumN using recursive to try satisfied array
func tryElementSumN(arr []int, currentSum, n, maxLen int, result chan []int) {
	var (
		lastElementValue int
	)

	if currentSum == n {
		if len(arr) <= maxLen {
			newSlice := make([]int, len(arr))
			copy(newSlice, arr)
			result <- newSlice
		}
		return
	}

	if currentSum > n || len(arr) > maxLen {
		return
	}

	currentLen := len(arr)
	if currentLen == 0 {
		lastElementValue = 1
	} else {
		lastElementValue = arr[currentLen-1]
	}
	for i := lastElementValue; i <= n && currentLen+1 <= maxLen; i++ {
		newArr := append(arr, i)
		tryElementSumN(newArr, currentSum+i, n, maxLen, result)
	}

	if len(arr) == 0 && currentSum == 0 {
		close(result)
	}
}

// generateArraySumN will generate list of elements which has sum all elements equal to n, and it's array length <= maxLen
func generateArraySumN(n, maxLen int) [][]int {
	//return [][]int{{14, 6}}
	var (
		ch     = make(chan []int)
		result [][]int
	)

	go tryElementSumN([]int{}, 0, n, maxLen, ch)

	for arr := range ch {
		permutationArr := make([]int, len(arr))
		copy(permutationArr, arr)
		result = append(result, arr)

		for i := 1; nextPermutation(sort.IntSlice(permutationArr)); i++ {
			tmpArr := make([]int, len(permutationArr))
			copy(tmpArr, permutationArr)
			result = append(result, tmpArr)
		}
	}
	//fmt.Println(result)
	return result
}
