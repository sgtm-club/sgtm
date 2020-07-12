package sgtmpb

import (
	"fmt"
	"time"
)

func (p *Post) CanonicalURL() string {
	return fmt.Sprintf("/post/%d", p.ID)
}

func (p *Post) GoDuration() time.Duration {
	return time.Millisecond * time.Duration(p.Duration)
}

func (u *User) CanonicalURL() string {
	return fmt.Sprintf("/@%s", u.Slug)
}
