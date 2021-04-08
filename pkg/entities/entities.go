package entities

import "moul.io/sgtm/pkg/sgtmpb"

type UploadsByWeekDay struct {
	Weekday  int64
	Quantity int64
}

type PostByKind struct {
	Kind     sgtmpb.Post_Kind
	Quantity int64
}
