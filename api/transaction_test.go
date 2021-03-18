package api

/*
func Test_rawToTransaction(t *testing.T) {
	InitMetrics()
	type args struct {
		c      *Client
		txRaw  TxResponse
		txLog  []LogFormat
		blocks map[uint64]shared.Block
	}
	tests := []struct {
		name     string
		filename string
		args     args
		want     cStruct.OutResp
		wantErr  bool
	}{
		{
			name:     "test2",
			filename: "./test/test2.json",
			args: args{
				c:      NewClient("", "", zaptest.NewLogger(t), nil, 0),
				blocks: map[uint64]shared.Block{340209: {}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			a, err := os.Open(tt.filename)
			log.Println("err", err)
			defer a.Close()
			b := json.NewDecoder(a)
			result := &GetTxSearchResponse{}
			err = b.Decode(result)

			readr := strings.NewReader("")
			dec := json.NewDecoder(readr)

			for _, txRaw := range result.Result.Txs {
				readr.Reset(txRaw.TxResult.Log)
				lf := []LogFormat{}
				err := dec.Decode(&lf)

				got, err := rawToTransaction(tt.args.c, txRaw, lf, tt.args.blocks)
				if (err != nil) != tt.wantErr {
					t.Errorf("rawToTransaction() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("rawToTransaction() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
*/
