package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPool_CanSwapTo(t *testing.T) {
	t.Parallel()
	type fields struct {
		Info PoolInfo
	}
	type args struct {
		address string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			"happy",
			fields{
				Info: PoolInfo{
					Tokens: []string{"token1", "token2"},
				},
			},
			args{
				address: "token1",
			},
			[]string{"token2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pool{
				Info: tt.fields.Info,
			}
			assert.Equalf(t, tt.want, p.CanSwapTo(tt.args.address), "CanSwapTo(%v)", tt.args.address)
		})
	}
}
