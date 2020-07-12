package sgtmpb

import "fmt"

func (p *Post) CanonicalURL() string {
	return fmt.Sprintf("/post/%d", p.ID)
}
