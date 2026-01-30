package crawler

import (
	"context"
	"testing"
)

func TestCrawler3_GetClassInfosForUndergraduate(t *testing.T) {
	c := NewClassCrawler3(new(MockProxyGetter))

	type args struct {
		ctx      context.Context
		stuID    string
		year     string
		semester string
		cookie   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				ctx:      context.Background(),
				stuID:    "testID",
				year:     "2025",
				semester: "1",
				cookie:   "bzb_jsxsd=54E11D4F5AC55766CB75E5B242A20339; __root_domain_v=.ccnu.edu.cn; _qddaz=QD.813351615258360; SERVERID=pcjw3; bzb_njw=A39CCD3688E8B687650C736A225B8202; SERVERIDgld=pc2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, err := c.GetClassInfosForUndergraduate(tt.args.ctx, tt.args.stuID, tt.args.year, tt.args.semester, tt.args.cookie)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClassInfosForUndergraduate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for _, v := range got {
				t.Log(*v)
			}
			for _, v := range got1 {
				t.Log(*v)
			}
			t.Logf("got2: %v", got2)

		})
	}
}
