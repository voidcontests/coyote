package version

import "log/slog"

var GIT_COMMIT string
var GIT_BRANCH string

var CommitAttr = slog.Attr{Key: "commit", Value: slog.StringValue(GIT_COMMIT)}
var BranchAttr = slog.Attr{Key: "branch", Value: slog.StringValue(GIT_BRANCH)}
