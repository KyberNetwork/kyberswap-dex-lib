package bruteforce

import (
	"reflect"
	"testing"
)

type testCase struct {
	n      int
	maxLen int
	want   [][]int
}

func TestGenerateArraySumN(t *testing.T) {

	cases := []testCase{
		{n: 5, maxLen: 2, want: [][]int{{1, 4}, {4, 1}, {2, 3}, {3, 2}, {5}}},
		{n: 5, maxLen: 3, want: [][]int{{1, 1, 3}, {1, 3, 1}, {3, 1, 1}, {1, 2, 2}, {2, 1, 2}, {2, 2, 1}, {1, 4}, {4, 1}, {2, 3}, {3, 2}, {5}}},
		{n: 5, maxLen: 5, want: [][]int{{1, 1, 1, 1, 1}, {1, 1, 1, 2}, {1, 1, 2, 1}, {1, 2, 1, 1}, {2, 1, 1, 1}, {1, 1, 3}, {1, 3, 1}, {3, 1, 1}, {1, 2, 2}, {2, 1, 2}, {2, 2, 1}, {1, 4}, {4, 1}, {2, 3}, {3, 2}, {5}}},
	}

	for _, sample := range cases {
		got := generateArraySumN(sample.n, sample.maxLen)
		if !reflect.DeepEqual(got, sample.want) {
			t.Errorf("got %+v want %+v", got, sample.want)
		}
	}

}
