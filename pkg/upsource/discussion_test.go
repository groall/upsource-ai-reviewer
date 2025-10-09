package upsource

import (
	"testing"
)

func Test_findRangeInFileContent(t *testing.T) {
	type args struct {
		fileContent string
		line        int
	}
	tests := []struct {
		name          string
		args          args
		wantStart     int32
		wantEnd       int32
		wantErr       bool
		expectedError string
	}{
		{
			name: "Test with a valid line number in the middle of the file",
			args: args{
				fileContent: "line 1\nline 2\nline 3",
				line:        2,
			},
			wantStart: 7,
			wantEnd:   13,
			wantErr:   false,
		},
		{
			name: "Test with the first line",
			args: args{
				fileContent: "line 1\nline 2\nline 3",
				line:        1,
			},
			wantStart: 0,
			wantEnd:   6,
			wantErr:   false,
		},
		{
			name: "Test with the last line",
			args: args{
				fileContent: "line 1\nline 2\nline 3",
				line:        3,
			},
			wantStart: 14,
			wantEnd:   20,
			wantErr:   false,
		},
		{
			name: "Test with a line number less than or equal to 0",
			args: args{
				fileContent: "line 1\nline 2\nline 3",
				line:        0,
			},
			wantErr:       true,
			expectedError: "line number must be positive: 0",
		},
		{
			name: "Test with a line number greater than the number of lines in the file",
			args: args{
				fileContent: "line 1\nline 2\nline 3",
				line:        4,
			},
			wantErr:       true,
			expectedError: "line 4 does not exist in a file with 3 lines",
		},
		{
			name: "Test with an empty line",
			args: args{
				fileContent: "line 1\n\nline 3",
				line:        2,
			},
			wantErr:       true,
			expectedError: "line 2 is empty",
		},
		{
			name: "Test with a file content that has no newlines",
			args: args{
				fileContent: "this is a single line",
				line:        1,
			},
			wantStart: 0,
			wantEnd:   21,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd, err := findRangeInFileContent(tt.args.fileContent, tt.args.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("findRangeInFileContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.expectedError {
				t.Errorf("findRangeInFileContent() error = %v, expectedError %v", err.Error(), tt.expectedError)
			}
			if gotStart != tt.wantStart {
				t.Errorf("findRangeInFileContent() gotStart = %v, want %v", gotStart, tt.wantStart)
			}
			if gotEnd != tt.wantEnd {
				t.Errorf("findRangeInFileContent() gotEnd = %v, want %v", gotEnd, tt.wantEnd)
			}
		})
	}
}
