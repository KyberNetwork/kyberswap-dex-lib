package getcustomroute

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_useCase_getSources(t *testing.T) {
	type fields struct {
		config Config
	}
	type args struct {
		includedSources     []string
		excludedSources     []string
		onlyScalableSources bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			name: "default",
			fields: fields{
				config: Config{
					AvailableSources: []string{"a", "b", "c"},
				},
			},
			args: args{
				includedSources:     nil,
				excludedSources:     nil,
				onlyScalableSources: false,
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "default onlyScalableSources",
			fields: fields{
				config: Config{
					AvailableSources:  []string{"a", "b", "c"},
					UnscalableSources: []string{"a"},
				},
			},
			args: args{
				includedSources:     nil,
				excludedSources:     nil,
				onlyScalableSources: true,
			},
			want: []string{"b", "c"},
		},
		{
			name: "included",
			fields: fields{
				config: Config{
					AvailableSources: []string{"a", "b", "c"},
				},
			},
			args: args{
				includedSources:     []string{"a", "b"},
				excludedSources:     nil,
				onlyScalableSources: false,
			},
			want: []string{"a", "b"},
		},
		{
			name: "included but onlyScalableSources",
			fields: fields{
				config: Config{
					AvailableSources:  []string{"a", "b", "c"},
					UnscalableSources: []string{"a"},
				},
			},
			args: args{
				includedSources:     []string{"a", "b"},
				excludedSources:     nil,
				onlyScalableSources: true,
			},
			want: []string{"b"},
		},
		{
			name: "excluded",
			fields: fields{
				config: Config{
					AvailableSources: []string{"a", "b", "c"},
				},
			},
			args: args{
				includedSources:     nil,
				excludedSources:     []string{"a", "b"},
				onlyScalableSources: false,
			},
			want: []string{"c"},
		},
		{
			name: "excluded and onlyScalableSources",
			fields: fields{
				config: Config{
					AvailableSources:  []string{"a", "b", "c"},
					UnscalableSources: []string{"a"},
				},
			},
			args: args{
				includedSources:     nil,
				excludedSources:     []string{"b"},
				onlyScalableSources: true,
			},
			want: []string{"c"},
		},
		{
			name: "included and excluded",
			fields: fields{
				config: Config{
					AvailableSources: []string{"a", "b", "c"},
				},
			},
			args: args{
				includedSources:     []string{"a", "b"},
				excludedSources:     []string{"a"},
				onlyScalableSources: false,
			},
			want: []string{"b"},
		},
		{
			name: "included and excluded and onlyScalableSources",
			fields: fields{
				config: Config{
					AvailableSources:  []string{"a", "b", "c"},
					UnscalableSources: []string{"b"},
				},
			},
			args: args{
				includedSources:     []string{"a", "b", "c"},
				excludedSources:     []string{"a"},
				onlyScalableSources: true,
			},
			want: []string{"c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &useCase{
				config: tt.fields.config,
			}
			got := u.getSources("", tt.args.includedSources, tt.args.excludedSources, tt.args.onlyScalableSources)
			assert.ElementsMatch(t, got, tt.want)
		})
	}
}
