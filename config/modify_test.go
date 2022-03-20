package config

import (
	"reflect"
	"testing"
)

func Test_sortOperationsDRA(t *testing.T) {
	type args struct {
		ops []*OperationConfig
	}
	tests := []struct {
		name string
		args args
		want []*OperationConfig
	}{
		{
			name: "empty",
			args: args{
				ops: []*OperationConfig{},
			},
			want: []*OperationConfig{},
		},
		{
			// sort based on operation type
			name: "delete_replace_add",
			args: args{
				ops: []*OperationConfig{
					{
						Operation: "add",
					},
					{
						Operation: "delete",
					},
					{
						Operation: "replace",
					},
					{
						Operation: "delete",
					},
				},
			},
			want: []*OperationConfig{
				{
					Operation: "delete",
				},
				{
					Operation: "delete",
				},
				{
					Operation: "replace",
				},
				{
					Operation: "add",
				},
			},
		},
		{
			// sort deletes based on type
			name: "sort_deletes",
			args: args{
				ops: []*OperationConfig{
					{
						Operation: "delete",
						NH:        new(nhEntry),
					},
					{
						Operation: "delete",
						NHG:       new(nhgEntry),
					},
					{
						Operation: "delete",
						IPv4:      new(ipv4v6Entry),
					},
					{
						Operation: "delete",
						IPv6:      new(ipv4v6Entry),
					},
				},
			},
			want: []*OperationConfig{
				{
					Operation: "delete",
					IPv4:      new(ipv4v6Entry),
				},
				{
					Operation: "delete",
					IPv6:      new(ipv4v6Entry),
				},
				{
					Operation: "delete",
					NHG:       new(nhgEntry),
				},
				{
					Operation: "delete",
					NH:        new(nhEntry),
				},
			},
		},
		{
			// sort replaces based on type
			name: "sort_replaces",
			args: args{
				ops: []*OperationConfig{
					{
						Operation: "replace",
						NH:        new(nhEntry),
					},
					{
						Operation: "replace",
						IPv4:      new(ipv4v6Entry),
					},
					{
						Operation: "replace",
						NHG:       new(nhgEntry),
					},
					{
						Operation: "replace",
						IPv6:      new(ipv4v6Entry),
					},
				},
			},
			want: []*OperationConfig{
				{
					Operation: "replace",
					NH:        new(nhEntry),
				},
				{
					Operation: "replace",
					NHG:       new(nhgEntry),
				},
				{
					Operation: "replace",
					IPv4:      new(ipv4v6Entry),
				},
				{
					Operation: "replace",
					IPv6:      new(ipv4v6Entry),
				},
			},
		},
		{
			// sort adds based on type
			name: "sort_adds",
			args: args{
				ops: []*OperationConfig{
					{
						Operation: "add",
						NH:        new(nhEntry),
					},
					{
						Operation: "add",
						IPv4:      new(ipv4v6Entry),
					},
					{
						Operation: "add",
						NHG:       new(nhgEntry),
					},
					{
						Operation: "add",
						IPv6:      new(ipv4v6Entry),
					},
				},
			},
			want: []*OperationConfig{
				{
					Operation: "add",
					NH:        new(nhEntry),
				},
				{
					Operation: "add",
					NHG:       new(nhgEntry),
				},
				{
					Operation: "add",
					IPv4:      new(ipv4v6Entry),
				},
				{
					Operation: "add",
					IPv6:      new(ipv4v6Entry),
				},
			},
		},
		{
			// sort deletes and adds based on type
			name: "sort_deletes_and_adds",
			args: args{
				ops: []*OperationConfig{
					{
						Operation: "add",
						NH:        new(nhEntry),
					},
					{
						Operation: "add",
						IPv4:      new(ipv4v6Entry),
					},
					{
						Operation: "add",
						NHG:       new(nhgEntry),
					},
					{
						Operation: "add",
						IPv6:      new(ipv4v6Entry),
					},
					{
						Operation: "delete",
						NH:        new(nhEntry),
					},
					{
						Operation: "delete",
						NHG:       new(nhgEntry),
					},
					{
						Operation: "delete",
						IPv4:      new(ipv4v6Entry),
					},
					{
						Operation: "delete",
						IPv6:      new(ipv4v6Entry),
					},
				},
			},
			want: []*OperationConfig{
				{
					Operation: "delete",
					IPv4:      new(ipv4v6Entry),
				},
				{
					Operation: "delete",
					IPv6:      new(ipv4v6Entry),
				},
				{
					Operation: "delete",
					NHG:       new(nhgEntry),
				},
				{
					Operation: "delete",
					NH:        new(nhEntry),
				},
				{
					Operation: "add",
					NH:        new(nhEntry),
				},
				{
					Operation: "add",
					NHG:       new(nhgEntry),
				},
				{
					Operation: "add",
					IPv4:      new(ipv4v6Entry),
				},
				{
					Operation: "add",
					IPv6:      new(ipv4v6Entry),
				},
			},
		},
		{
			// sort deletes, replaces and adds based on type
			name: "sort_deletes_replaces_and_adds",
			args: args{
				ops: []*OperationConfig{
					{
						Operation: "add",
						NH:        new(nhEntry),
					},
					{
						Operation: "add",
						IPv4:      new(ipv4v6Entry),
					},
					{
						Operation: "add",
						NHG:       new(nhgEntry),
					},
					{
						Operation: "add",
						IPv6:      new(ipv4v6Entry),
					},
					{
						Operation: "delete",
						NH:        new(nhEntry),
					},
					{
						Operation: "delete",
						NHG:       new(nhgEntry),
					},
					{
						Operation: "delete",
						IPv4:      new(ipv4v6Entry),
					},
					{
						Operation: "delete",
						IPv6:      new(ipv4v6Entry),
					},
					{
						Operation: "replace",
						NH:        new(nhEntry),
					},
					{
						Operation: "replace",
						IPv4:      new(ipv4v6Entry),
					},
					{
						Operation: "replace",
						NHG:       new(nhgEntry),
					},
					{
						Operation: "replace",
						IPv6:      new(ipv4v6Entry),
					},
				},
			},
			want: []*OperationConfig{
				{
					Operation: "delete",
					IPv4:      new(ipv4v6Entry),
				},
				{
					Operation: "delete",
					IPv6:      new(ipv4v6Entry),
				},
				{
					Operation: "delete",
					NHG:       new(nhgEntry),
				},
				{
					Operation: "delete",
					NH:        new(nhEntry),
				},
				{
					Operation: "replace",
					NH:        new(nhEntry),
				},
				{
					Operation: "replace",
					NHG:       new(nhgEntry),
				},
				{
					Operation: "replace",
					IPv4:      new(ipv4v6Entry),
				},
				{
					Operation: "replace",
					IPv6:      new(ipv4v6Entry),
				},
				{
					Operation: "add",
					NH:        new(nhEntry),
				},
				{
					Operation: "add",
					NHG:       new(nhgEntry),
				},
				{
					Operation: "add",
					IPv4:      new(ipv4v6Entry),
				},
				{
					Operation: "add",
					IPv6:      new(ipv4v6Entry),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if sortOperationsDRA(tt.args.ops); !reflect.DeepEqual(tt.args.ops, tt.want) {
				t.Errorf("sortOperations() = %v, want %v", tt.args.ops, tt.want)
			}
		})
	}
}
