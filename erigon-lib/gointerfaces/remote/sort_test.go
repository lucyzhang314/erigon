package remote_test

import (
	"sort"
	"testing"

	"github.com/ledgerwatch/erigon-lib/gointerfaces/remote"
	"github.com/ledgerwatch/erigon-lib/gointerfaces/types"
	"github.com/stretchr/testify/assert"
)

func TestSort(t *testing.T) {
	tests := []struct {
		name string
		got  *remote.NodesInfoReply
		want *remote.NodesInfoReply
	}{
		{
			name: "sort by name",
			got: &remote.NodesInfoReply{
				NodesInfo: []*types.NodeInfoReply{
					{Name: "b", Enode: "c"},
					{Name: "a", Enode: "d"},
				},
			},
			want: &remote.NodesInfoReply{
				NodesInfo: []*types.NodeInfoReply{
					{Name: "a", Enode: "d"},
					{Name: "b", Enode: "c"},
				},
			},
		},
		{
			name: "sort by enode",
			got: &remote.NodesInfoReply{
				NodesInfo: []*types.NodeInfoReply{
					{Name: "a", Enode: "d"},
					{Name: "a", Enode: "c"},
				},
			},
			want: &remote.NodesInfoReply{
				NodesInfo: []*types.NodeInfoReply{
					{Name: "a", Enode: "c"},
					{Name: "a", Enode: "d"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sort.Slice(tt.got.NodesInfo, func(i, j int) bool {
				return remote.NodeInfoReplyLess(tt.got.NodesInfo[i], tt.got.NodesInfo[j])
			})
			assert.Equal(t, tt.want, tt.got)
		})
	}
}
