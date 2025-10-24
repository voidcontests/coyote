package version

import "log/slog"

var Commit string
var Branch string

var CommitAttr = slog.Attr{Key: "commit", Value: slog.StringValue(Commit)}
var BranchAttr = slog.Attr{Key: "branch", Value: slog.StringValue(Branch)}
