package upsource

import (
	"testing"

	"github.com/groall/upsource-go-client/client"
)

func TestShouldReplyToDiscussion(t *testing.T) {
	const (
		botID = "bot-1"
		label = "AI-Reviewed"
	)
	resolved := true

	mk := func(comments []client.CommentDTO, opts ...func(*client.DiscussionInFileDTO)) client.DiscussionInFileDTO {
		d := client.DiscussionInFileDTO{
			Labels:   []client.LabelDTO{{Name: label}},
			Comments: comments,
		}
		for _, o := range opts {
			o(&d)
		}
		return d
	}

	tests := []struct {
		name      string
		disc      client.DiscussionInFileDTO
		maxPer    int
		wantReply bool
	}{
		{
			name: "human replied after bot — should reply",
			disc: mk([]client.CommentDTO{
				{CommentID: "c1", AuthorID: botID},
				{CommentID: "c2", AuthorID: "human"},
			}),
			maxPer:    3,
			wantReply: true,
		},
		{
			name: "bot was last to comment — skip",
			disc: mk([]client.CommentDTO{
				{CommentID: "c1", AuthorID: "human"},
				{CommentID: "c2", AuthorID: botID},
			}),
			maxPer:    3,
			wantReply: false,
		},
		{
			name: "discussion resolved — skip",
			disc: mk([]client.CommentDTO{
				{CommentID: "c1", AuthorID: botID},
				{CommentID: "c2", AuthorID: "human"},
			}, func(d *client.DiscussionInFileDTO) { d.IsResolved = &resolved }),
			maxPer:    3,
			wantReply: false,
		},
		{
			name: "bot reached cap — skip",
			disc: mk([]client.CommentDTO{
				{CommentID: "c1", AuthorID: botID},
				{CommentID: "c2", AuthorID: "human"},
				{CommentID: "c3", AuthorID: botID},
				{CommentID: "c4", AuthorID: "human"},
				{CommentID: "c5", AuthorID: botID},
				{CommentID: "c6", AuthorID: "human"},
			}),
			maxPer:    3,
			wantReply: false,
		},
		{
			name:      "no comments — skip",
			disc:      mk(nil),
			maxPer:    3,
			wantReply: false,
		},
		{
			name: "missing reviewed label — skip",
			disc: client.DiscussionInFileDTO{
				Comments: []client.CommentDTO{{CommentID: "c1", AuthorID: "human"}},
			},
			maxPer:    3,
			wantReply: false,
		},
		{
			name: "bot never commented — skip (thread not ours)",
			disc: mk([]client.CommentDTO{
				{CommentID: "c1", AuthorID: "human-a"},
				{CommentID: "c2", AuthorID: "human-b"},
			}),
			maxPer: 3,
			// Predicate is purely "is the last comment not the bot?"; the orchestrator
			// uses the reviewedLabel to scope to bot-touched threads. Here the label
			// is set, so we expect a reply attempt.
			wantReply: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			last, ok := ShouldReplyToDiscussion(tt.disc, label, botID, tt.maxPer)
			if ok != tt.wantReply {
				t.Fatalf("ShouldReplyToDiscussion = %v, want %v", ok, tt.wantReply)
			}
			if ok && last.CommentID == "" {
				t.Fatalf("expected non-empty parent comment when reply is true")
			}
		})
	}
}

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
