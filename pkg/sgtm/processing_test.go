package sgtm

import (
	"io"
	"log"
	"os"
	"reflect"
	"testing"
)

func TestExtractAbletonTrackInfos(t *testing.T) {
	file, err := os.Open("../../test/TestTrack.als")
	if err != nil {
		log.Fatal(err)
	}

	type args struct {
		fileReader io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    *TrackSourceFile
		wantErr bool
	}{
		{"should parse a correct ableton file",
			args{fileReader: file},
			&TrackSourceFile{
				Daw:     "Ableton Live 10.0.4",
				Tracks:  26,
				Plugins: []string{"Serum", "BalanceSPTeufelsbergReverb", "FabFilter Pro-Q 2", "Auburn Sounds Graillon 2"},
			},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractAbletonTrackInfos(tt.args.fileReader)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractAbletonTrackInfos() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractAbletonTrackInfos() got = %v, want %v", got, tt.want)
			}
		})
	}
}
